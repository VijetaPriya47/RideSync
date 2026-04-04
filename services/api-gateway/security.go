package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"ride-sharing/shared/authjwt"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
)

type ctxKey int

const (
	ctxSubject ctxKey = iota
	ctxRole
	ctxEmail
)

func withAuth(ctx context.Context, sub, role, email string) context.Context {
	ctx = context.WithValue(ctx, ctxSubject, sub)
	ctx = context.WithValue(ctx, ctxRole, role)
	ctx = context.WithValue(ctx, ctxEmail, email)
	return ctx
}

func authFromRequest(r *http.Request) (sub, role, email string, ok bool) {
	sub, _ = r.Context().Value(ctxSubject).(string)
	role, _ = r.Context().Value(ctxRole).(string)
	email, _ = r.Context().Value(ctxEmail).(string)
	return sub, role, email, sub != ""
}

func isPublicPath(path string) bool {
	switch {
	case path == "/webhook/stripe":
		return true
	case path == "/health":
		return true
	case strings.HasPrefix(path, "/ws/"):
		return true
	case strings.HasPrefix(path, "/api/auth/"):
		return true
	default:
		return false
	}
}

func mutatingMethod(m string) bool {
	switch m {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	default:
		return false
	}
}

func applyCanonicalUserID(bodyUserID *string, sub string, authed bool) bool {
	if !authed || sub == "" {
		return false
	}
	if *bodyUserID != "" && *bodyUserID != sub {
		return false
	}
	*bodyUserID = sub
	return true
}

func rbacAllowed(role, path, method string) bool {
	if strings.HasPrefix(path, "/api/finance/dashboard") {
		return role == authjwt.RoleBusiness || role == authjwt.RoleAdmin
	}
	if path == "/api/finance/me" && method == http.MethodGet {
		return role == authjwt.RoleCustomer
	}
	if strings.HasPrefix(path, "/api/admin/") {
		return role == authjwt.RoleAdmin
	}
	if strings.HasPrefix(path, "/trip/") || path == "/api/trips/book" {
		return role == authjwt.RoleCustomer
	}
	return true
}

func chainHTTP(middlewares []func(http.Handler) http.Handler, final http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		final = middlewares[i](final)
	}
	return final
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func jwtMiddleware(next http.Handler) http.Handler {
	secret := []byte(env.GetString("JWT_SECRET", "dev-insecure-change-me"))
	iss := env.GetString("JWT_ISSUER", "ridesync-auth")
	aud := env.GetString("JWT_AUDIENCE", "ridesync-gateway")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		h := r.Header.Get("Authorization")
		parts := strings.Fields(h)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			writeJSONError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		raw := parts[1]
		claims, err := authjwt.Parse(secret, iss, aud, raw)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := withAuth(r.Context(), claims.Subject, claims.Role, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func rbacMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		_, role, _, ok := authFromRequest(r)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !rbacAllowed(role, r.URL.Path, r.Method) {
			writeJSONError(w, http.StatusForbidden, "forbidden for this role")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.code == 0 {
		s.code = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) status() int {
	if s.code == 0 {
		return http.StatusOK
	}
	return s.code
}

func auditMiddleware(rmq *messaging.RabbitMQ) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mutatingMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			rec := &statusRecorder{ResponseWriter: w, code: 0}
			next.ServeHTTP(rec, r)

			sub, role, _, _ := authFromRequest(r)
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			if host == "" {
				host = r.RemoteAddr
			}
			pl := messaging.AuditLogPayload{
				Method:      r.Method,
				Path:        r.URL.Path,
				ActorUserID: sub,
				Role:        role,
				IP:          host,
				TS:          time.Now().UTC().Format(time.RFC3339),
				StatusCode:  rec.status(),
			}
			data, err := json.Marshal(pl)
			if err != nil {
				return
			}
			msg := contracts.AmqpMessage{OwnerID: sub, Data: data}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := rmq.PublishMessage(ctx, contracts.AuditEventWrite, msg); err != nil {
					log.Printf("audit publish: %v", err)
				}
			}()
		})
	}
}
