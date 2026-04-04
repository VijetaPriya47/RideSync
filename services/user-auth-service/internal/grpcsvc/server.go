package grpcsvc

import (
	"context"
	"log"
	"os"
	"time"

	"ride-sharing/services/user-auth-service/internal/repo"
	pb "ride-sharing/shared/proto/auth"
	"ride-sharing/shared/authjwt"
	"ride-sharing/shared/env"

	"google.golang.org/api/idtoken"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedUserAuthServiceServer
	Repo *repo.Repo
}

func (s *Server) jwtForUser(u *repo.User) (string, error) {
	secret := []byte(env.GetString("JWT_SECRET", "dev-insecure-change-me"))
	iss := env.GetString("JWT_ISSUER", "ridesync-auth")
	aud := env.GetString("JWT_AUDIENCE", "ridesync-gateway")
	ttl := 48 * time.Hour
	if s := os.Getenv("JWT_TTL"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			ttl = d
		}
	}
	return authjwt.Sign(secret, iss, aud, u.ID, u.Email, u.Role, ttl)
}

func (s *Server) summary(u *repo.User) *pb.UserSummary {
	return &pb.UserSummary{Id: u.ID, Email: u.Email, Role: u.Role}
}

func (s *Server) LoginLocal(ctx context.Context, req *pb.LoginLocalRequest) (*pb.LoginLocalResponse, error) {
	u, err := s.Repo.GetUserByEmail(ctx, req.GetEmail())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	if u == nil || u.PasswordHash == nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if u.Role == authjwt.RoleCustomer {
		return nil, status.Error(codes.PermissionDenied, "use Google sign-in for customer accounts")
	}
	if err := repo.VerifyPassword(*u.PasswordHash, req.GetPassword()); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	tok, err := s.jwtForUser(u)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.LoginLocalResponse{Jwt: tok, User: s.summary(u)}, nil
}

func (s *Server) GoogleVerify(ctx context.Context, req *pb.GoogleVerifyRequest) (*pb.GoogleVerifyResponse, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		return nil, status.Error(codes.FailedPrecondition, "GOOGLE_CLIENT_ID not configured")
	}
	payload, err := idtoken.Validate(ctx, req.GetIdToken(), clientID)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid id token: %v", err)
	}
	email, _ := payload.Claims["email"].(string)
	if email == "" {
		return nil, status.Error(codes.Unauthenticated, "email claim missing")
	}
	u, err := s.Repo.UpsertGoogleCustomer(ctx, payload.Subject, email)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "%v", err)
	}
	tok, err := s.jwtForUser(u)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.GoogleVerifyResponse{Jwt: tok, User: s.summary(u)}, nil
}

func (s *Server) RegisterBusiness(ctx context.Context, req *pb.RegisterBusinessRequest) (*pb.RegisterBusinessResponse, error) {
	admin, err := s.Repo.GetUserByID(ctx, req.GetAdminUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	if admin == nil || admin.Role != authjwt.RoleAdmin {
		return nil, status.Error(codes.PermissionDenied, "admin only")
	}
	u, err := s.Repo.CreateLocalUser(ctx, req.GetEmail(), req.GetPassword(), authjwt.RoleBusiness)
	if err != nil {
		return nil, status.Errorf(codes.AlreadyExists, "%v", err)
	}
	return &pb.RegisterBusinessResponse{User: s.summary(u)}, nil
}

func (s *Server) RegisterAdmin(ctx context.Context, req *pb.RegisterAdminRequest) (*pb.RegisterAdminResponse, error) {
	admin, err := s.Repo.GetUserByID(ctx, req.GetAdminUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	if admin == nil || admin.Role != authjwt.RoleAdmin {
		return nil, status.Error(codes.PermissionDenied, "admin only")
	}
	u, err := s.Repo.CreateLocalUser(ctx, req.GetEmail(), req.GetPassword(), authjwt.RoleAdmin)
	if err != nil {
		return nil, status.Errorf(codes.AlreadyExists, "%v", err)
	}
	return &pb.RegisterAdminResponse{User: s.summary(u)}, nil
}

func (s *Server) RequestPasswordReset(ctx context.Context, req *pb.RequestPasswordResetRequest) (*pb.RequestPasswordResetResponse, error) {
	u, err := s.Repo.GetUserByEmail(ctx, req.GetEmail())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	if u == nil {
		// Do not reveal existence
		return &pb.RequestPasswordResetResponse{}, nil
	}
	if u.Role != authjwt.RoleBusiness {
		return &pb.RequestPasswordResetResponse{}, nil
	}
	raw, err := s.Repo.CreatePasswordResetToken(ctx, u.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	base := env.GetString("PUBLIC_GATEWAY_URL", "http://localhost:8081")
	log.Printf("[simulated email] password reset for %s: POST %s/api/auth/reset-password with token=%s", u.Email, base, raw)
	return &pb.RequestPasswordResetResponse{}, nil
}

func (s *Server) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordResponse, error) {
	if err := s.Repo.ResetPasswordWithToken(ctx, req.GetToken(), req.GetNewPassword()); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return &pb.ResetPasswordResponse{}, nil
}

func (s *Server) ListAuditLogs(ctx context.Context, req *pb.ListAuditLogsRequest) (*pb.ListAuditLogsResponse, error) {
	var before *time.Time
	if ts := req.GetBeforeTsRfc3339(); ts != "" {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid before_ts")
		}
		before = &t
	}
	entries, err := s.Repo.ListAuditLogs(ctx, req.GetLimit(), before)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.ListAuditLogsResponse{Entries: entries}, nil
}

func (s *Server) InsertAuditLog(ctx context.Context, req *pb.InsertAuditLogRequest) (*pb.InsertAuditLogResponse, error) {
	err := s.Repo.InsertAuditLog(ctx, req.GetMethod(), req.GetPath(), req.GetActorUserId(), req.GetRole(), req.GetIp(), req.GetDetailJson())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.InsertAuditLogResponse{}, nil
}
