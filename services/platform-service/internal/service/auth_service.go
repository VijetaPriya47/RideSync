package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"ride-sharing/services/platform-service/internal/domain"
	"ride-sharing/services/platform-service/internal/infrastructure/repository"
	"ride-sharing/shared/authjwt"
	"ride-sharing/shared/env"
	pb "ride-sharing/shared/proto/auth"

	"google.golang.org/api/idtoken"
)

type authService struct {
	repo domain.UserRepository
}

// NewAuthService returns an AuthService backed by the given repository.
func NewAuthService(repo domain.UserRepository) domain.AuthService {
	return &authService{repo: repo}
}

func (s *authService) EnsureBootstrap(ctx context.Context) error {
	return s.repo.EnsureSuperAdmin(ctx,
		env.GetString("SUPER_ADMIN_EMAIL", "vijeta.admin@ridesync.com"),
		env.GetString("SUPER_ADMIN_PASSWORD", "change-me"),
	)
}

func (s *authService) jwtForUser(u *domain.User) (string, error) {
	secret := []byte(env.GetString("JWT_SECRET", "dev-insecure-change-me"))
	iss := env.GetString("JWT_ISSUER", "ridesync-auth")
	aud := env.GetString("JWT_AUDIENCE", "ridesync-gateway")
	ttl := 48 * time.Hour
	if v := os.Getenv("JWT_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		}
	}
	return authjwt.Sign(secret, iss, aud, u.ID, u.Email, u.Role, ttl)
}

func (s *authService) LoginLocal(ctx context.Context, email, password string) (string, *domain.User, error) {
	u, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}
	if u == nil || u.PasswordHash == nil {
		return "", nil, ErrInvalidCredentials
	}
	if u.Role == authjwt.RoleCustomer {
		return "", nil, ErrCustomerUseGoogle
	}
	if err := repository.ComparePasswordHash(*u.PasswordHash, password); err != nil {
		return "", nil, ErrInvalidCredentials
	}
	tok, err := s.jwtForUser(u)
	if err != nil {
		return "", nil, err
	}
	return tok, u, nil
}

// Sentinel errors for gRPC mapping.
var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrCustomerUseGoogle    = errors.New("use Google sign-in for customer accounts")
	ErrGoogleNotConfigured  = errors.New("GOOGLE_CLIENT_ID not configured")
	ErrEmailClaimMissing    = errors.New("email claim missing")
	ErrAdminOnly            = errors.New("admin only")
	ErrRegisterUserConflict = errors.New("user registration conflict")
	ErrInvalidAuditBefore   = errors.New("invalid before_ts")
	ErrGoogleUpsertDenied   = errors.New("google account upsert denied")
)

func (s *authService) GoogleVerify(ctx context.Context, idToken string) (string, *domain.User, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		return "", nil, ErrGoogleNotConfigured
	}
	payload, err := idtoken.Validate(ctx, idToken, clientID)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrInvalidCredentials, err)
	}
	email, _ := payload.Claims["email"].(string)
	if email == "" {
		return "", nil, ErrEmailClaimMissing
	}
	u, err := s.repo.UpsertGoogleCustomer(ctx, payload.Subject, email)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrGoogleUpsertDenied, err)
	}
	tok, err := s.jwtForUser(u)
	if err != nil {
		return "", nil, err
	}
	return tok, u, nil
}

func (s *authService) RegisterBusiness(ctx context.Context, adminUserID, email, password string) (*domain.User, error) {
	admin, err := s.repo.GetUserByID(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	if admin == nil || admin.Role != authjwt.RoleAdmin {
		return nil, ErrAdminOnly
	}
	u, err := s.repo.CreateLocalUser(ctx, email, password, authjwt.RoleBusiness)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegisterUserConflict, err)
	}
	return u, nil
}

func (s *authService) RegisterAdmin(ctx context.Context, adminUserID, email, password string) (*domain.User, error) {
	admin, err := s.repo.GetUserByID(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	if admin == nil || admin.Role != authjwt.RoleAdmin {
		return nil, ErrAdminOnly
	}
	u, err := s.repo.CreateLocalUser(ctx, email, password, authjwt.RoleAdmin)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRegisterUserConflict, err)
	}
	return u, nil
}

func (s *authService) RequestPasswordReset(ctx context.Context, email string) error {
	u, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if u == nil || u.Role != authjwt.RoleBusiness {
		return nil
	}
	raw, err := s.repo.CreatePasswordResetToken(ctx, u.ID)
	if err != nil {
		return err
	}
	base := env.GetString("PUBLIC_GATEWAY_URL", "http://localhost:8081")
	log.Printf("[simulated email] password reset for %s: POST %s/api/auth/reset-password with token=%s", u.Email, base, raw)
	return nil
}

func (s *authService) ResetPassword(ctx context.Context, token, newPassword string) error {
	return s.repo.ResetPasswordWithToken(ctx, token, newPassword)
}

func (s *authService) ListAuditLogs(ctx context.Context, limit int32, beforeRFC3339 string) ([]*pb.AuditLogEntry, error) {
	var before *time.Time
	if beforeRFC3339 != "" {
		t, err := time.Parse(time.RFC3339, beforeRFC3339)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidAuditBefore, err)
		}
		before = &t
	}
	return s.repo.ListAuditLogs(ctx, limit, before)
}

func (s *authService) InsertAuditLog(ctx context.Context, method, path, actorID, role, ip, detailJSON string) error {
	return s.repo.InsertAuditLog(ctx, method, path, actorID, role, ip, detailJSON)
}
