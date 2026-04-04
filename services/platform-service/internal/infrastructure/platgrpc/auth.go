package platgrpc

import (
	"context"
	"errors"

	"ride-sharing/services/platform-service/internal/domain"
	"ride-sharing/services/platform-service/internal/service"
	pb "ride-sharing/shared/proto/auth"

	grpcstd "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authHandler struct {
	pb.UnimplementedUserAuthServiceServer
	svc domain.AuthService
}

// RegisterUserAuth registers UserAuthService on the gRPC server.
func RegisterUserAuth(server *grpcstd.Server, svc domain.AuthService) {
	pb.RegisterUserAuthServiceServer(server, &authHandler{svc: svc})
}

func userSummary(u *domain.User) *pb.UserSummary {
	if u == nil {
		return nil
	}
	return &pb.UserSummary{Id: u.ID, Email: u.Email, Role: u.Role}
}

func (h *authHandler) LoginLocal(ctx context.Context, req *pb.LoginLocalRequest) (*pb.LoginLocalResponse, error) {
	tok, u, err := h.svc.LoginLocal(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			return nil, status.Error(codes.Unauthenticated, err.Error())
		case errors.Is(err, service.ErrCustomerUseGoogle):
			return nil, status.Error(codes.PermissionDenied, err.Error())
		default:
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
	}
	return &pb.LoginLocalResponse{Jwt: tok, User: userSummary(u)}, nil
}

func (h *authHandler) GoogleVerify(ctx context.Context, req *pb.GoogleVerifyRequest) (*pb.GoogleVerifyResponse, error) {
	tok, u, err := h.svc.GoogleVerify(ctx, req.GetIdToken())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGoogleNotConfigured):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		case errors.Is(err, service.ErrInvalidCredentials):
			return nil, status.Errorf(codes.Unauthenticated, "%v", err)
		case errors.Is(err, service.ErrEmailClaimMissing):
			return nil, status.Error(codes.Unauthenticated, err.Error())
		case errors.Is(err, service.ErrGoogleUpsertDenied):
			return nil, status.Errorf(codes.PermissionDenied, "%v", err)
		default:
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
	}
	return &pb.GoogleVerifyResponse{Jwt: tok, User: userSummary(u)}, nil
}

func (h *authHandler) RegisterBusiness(ctx context.Context, req *pb.RegisterBusinessRequest) (*pb.RegisterBusinessResponse, error) {
	u, err := h.svc.RegisterBusiness(ctx, req.GetAdminUserId(), req.GetEmail(), req.GetPassword())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAdminOnly):
			return nil, status.Error(codes.PermissionDenied, err.Error())
		case errors.Is(err, service.ErrRegisterUserConflict):
			return nil, status.Errorf(codes.AlreadyExists, "%v", err)
		default:
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
	}
	return &pb.RegisterBusinessResponse{User: userSummary(u)}, nil
}

func (h *authHandler) RegisterAdmin(ctx context.Context, req *pb.RegisterAdminRequest) (*pb.RegisterAdminResponse, error) {
	u, err := h.svc.RegisterAdmin(ctx, req.GetAdminUserId(), req.GetEmail(), req.GetPassword())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAdminOnly):
			return nil, status.Error(codes.PermissionDenied, err.Error())
		case errors.Is(err, service.ErrRegisterUserConflict):
			return nil, status.Errorf(codes.AlreadyExists, "%v", err)
		default:
			return nil, status.Errorf(codes.Internal, "%v", err)
		}
	}
	return &pb.RegisterAdminResponse{User: userSummary(u)}, nil
}

func (h *authHandler) RequestPasswordReset(ctx context.Context, req *pb.RequestPasswordResetRequest) (*pb.RequestPasswordResetResponse, error) {
	if err := h.svc.RequestPasswordReset(ctx, req.GetEmail()); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.RequestPasswordResetResponse{}, nil
}

func (h *authHandler) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.ResetPasswordResponse, error) {
	if err := h.svc.ResetPassword(ctx, req.GetToken(), req.GetNewPassword()); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}
	return &pb.ResetPasswordResponse{}, nil
}

func (h *authHandler) ListAuditLogs(ctx context.Context, req *pb.ListAuditLogsRequest) (*pb.ListAuditLogsResponse, error) {
	entries, err := h.svc.ListAuditLogs(ctx, req.GetLimit(), req.GetBeforeTsRfc3339())
	if err != nil {
		if errors.Is(err, service.ErrInvalidAuditBefore) {
			return nil, status.Error(codes.InvalidArgument, "invalid before_ts")
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.ListAuditLogsResponse{Entries: entries}, nil
}

func (h *authHandler) InsertAuditLog(ctx context.Context, req *pb.InsertAuditLogRequest) (*pb.InsertAuditLogResponse, error) {
	if err := h.svc.InsertAuditLog(ctx, req.GetMethod(), req.GetPath(), req.GetActorUserId(), req.GetRole(), req.GetIp(), req.GetDetailJson()); err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.InsertAuditLogResponse{}, nil
}
