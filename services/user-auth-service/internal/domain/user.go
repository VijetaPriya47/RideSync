package domain

import (
	"context"
	"time"

	pb "ride-sharing/shared/proto/auth"
)

// User is an application-level user record.
type User struct {
	ID           string
	Email        string
	Role         string
	PasswordHash *string
	GoogleSub    *string
}

// UserRepository persists users, audit logs, and password reset tokens.
type UserRepository interface {
	EnsureSuperAdmin(ctx context.Context, email, plainPassword string) error
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateLocalUser(ctx context.Context, email, plainPassword, role string) (*User, error)
	UpsertGoogleCustomer(ctx context.Context, googleSub, email string) (*User, error)
	InsertAuditLog(ctx context.Context, method, path, actorID, role, ip, detailJSON string) error
	ListAuditLogs(ctx context.Context, limit int32, before *time.Time) ([]*pb.AuditLogEntry, error)
	CreatePasswordResetToken(ctx context.Context, userID string) (rawToken string, err error)
	ResetPasswordWithToken(ctx context.Context, rawToken, newPassword string) error
}

// AuthService covers authentication and admin provisioning.
type AuthService interface {
	EnsureBootstrap(ctx context.Context) error
	LoginLocal(ctx context.Context, email, password string) (jwt string, user *User, err error)
	GoogleVerify(ctx context.Context, idToken string) (jwt string, user *User, err error)
	RegisterBusiness(ctx context.Context, adminUserID, email, password string) (*User, error)
	RegisterAdmin(ctx context.Context, adminUserID, email, password string) (*User, error)
	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	ListAuditLogs(ctx context.Context, limit int32, beforeRFC3339 string) ([]*pb.AuditLogEntry, error)
	InsertAuditLog(ctx context.Context, method, path, actorID, role, ip, detailJSON string) error
}
