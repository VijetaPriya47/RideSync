package repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "ride-sharing/shared/proto/auth"
	"ride-sharing/shared/authjwt"
)

type Repo struct {
	Pool *pgxpool.Pool
}

type User struct {
	ID           string
	Email        string
	Role         string
	PasswordHash *string
	GoogleSub    *string
}

func (r *Repo) EnsureSuperAdmin(ctx context.Context, email, plainPassword string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || plainPassword == "" {
		return nil
	}
	var n int
	_ = r.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE email = $1`, email).Scan(&n)
	if n > 0 {
		return nil
	}
	hash, err := hashPassword(plainPassword)
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `
INSERT INTO users (email, password_hash, role) VALUES ($1, $2, 'admin')`, email, hash)
	return err
}

func (r *Repo) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	var gh, pw *string
	err := r.Pool.QueryRow(ctx, `
SELECT id::text, email, role, password_hash, google_sub FROM users WHERE id = $1::uuid`, id).
		Scan(&u.ID, &u.Email, &u.Role, &pw, &gh)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.PasswordHash = pw
	u.GoogleSub = gh
	return &u, nil
}

func (r *Repo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var u User
	var gh, pw *string
	err := r.Pool.QueryRow(ctx, `
SELECT id::text, email, role, password_hash, google_sub FROM users WHERE lower(email) = $1`, email).
		Scan(&u.ID, &u.Email, &u.Role, &pw, &gh)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.PasswordHash = pw
	u.GoogleSub = gh
	return &u, nil
}

func (r *Repo) CreateLocalUser(ctx context.Context, email, plainPassword, role string) (*User, error) {
	if role != authjwt.RoleBusiness && role != authjwt.RoleAdmin {
		return nil, fmt.Errorf("invalid role")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	hash, err := hashPassword(plainPassword)
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	_, err = r.Pool.Exec(ctx, `
INSERT INTO users (id, email, password_hash, role) VALUES ($1, $2, $3, $4)`, id, email, hash, role)
	if err != nil {
		return nil, err
	}
	return &User{ID: id.String(), Email: email, Role: role}, nil
}

func (r *Repo) UpsertGoogleCustomer(ctx context.Context, googleSub, email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if googleSub == "" || email == "" {
		return nil, fmt.Errorf("missing claims")
	}
	var u User
	err := r.Pool.QueryRow(ctx, `
SELECT id::text, email, role FROM users WHERE google_sub = $1`, googleSub).Scan(&u.ID, &u.Email, &u.Role)
	if err == nil {
		return &u, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	var existingID, existingEmail, existingRole string
	var existingSub *string
	err = r.Pool.QueryRow(ctx, `
SELECT id::text, email, role, google_sub FROM users WHERE lower(email) = $1`, email).
		Scan(&existingID, &existingEmail, &existingRole, &existingSub)
	if err == nil {
		if existingSub != nil && *existingSub == googleSub {
			return &User{ID: existingID, Email: existingEmail, Role: existingRole}, nil
		}
		return nil, fmt.Errorf("email already registered with a different login method")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	id := uuid.New()
	_, err = r.Pool.Exec(ctx, `
INSERT INTO users (id, email, role, google_sub) VALUES ($1, $2, 'customer', $3)`, id, email, googleSub)
	if err != nil {
		return nil, err
	}
	return &User{ID: id.String(), Email: email, Role: authjwt.RoleCustomer}, nil
}

func (r *Repo) InsertAuditLog(ctx context.Context, method, path, actorID, role, ip, detailJSON string) error {
	if detailJSON == "" {
		detailJSON = "{}"
	}
	var js json.RawMessage
	if err := json.Unmarshal([]byte(detailJSON), &js); err != nil {
		detailJSON = "{}"
	}
	_, err := r.Pool.Exec(ctx, `
INSERT INTO audit_logs (method, path, actor_user_id, role, ip, detail)
VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), NULLIF($5,''), $6::jsonb)`,
		method, path, actorID, role, ip, detailJSON)
	return err
}

func (r *Repo) ListAuditLogs(ctx context.Context, limit int32, before *time.Time) ([]*pb.AuditLogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows pgx.Rows
	var err error
	if before != nil {
		rows, err = r.Pool.Query(ctx, `
SELECT id::text, ts, method, path, COALESCE(actor_user_id,''), COALESCE(role,''), COALESCE(ip,''), detail::text
FROM audit_logs WHERE ts < $1 ORDER BY ts DESC LIMIT $2`, *before, limit)
	} else {
		rows, err = r.Pool.Query(ctx, `
SELECT id::text, ts, method, path, COALESCE(actor_user_id,''), COALESCE(role,''), COALESCE(ip,''), detail::text
FROM audit_logs ORDER BY ts DESC LIMIT $1`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*pb.AuditLogEntry
	for rows.Next() {
		var e pb.AuditLogEntry
		var ts time.Time
		if err := rows.Scan(&e.Id, &ts, &e.Method, &e.Path, &e.ActorUserId, &e.Role, &e.Ip, &e.DetailJson); err != nil {
			return nil, err
		}
		e.TsRfc3339 = ts.UTC().Format(time.RFC3339)
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (r *Repo) CreatePasswordResetToken(ctx context.Context, userID string) (rawToken string, err error) {
	raw := uuid.New().String() + uuid.New().String()
	h := sha256.Sum256([]byte(raw))
	tokenHash := hex.EncodeToString(h[:])
	exp := time.Now().Add(1 * time.Hour)
	_, err = r.Pool.Exec(ctx, `
INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1::uuid, $2, $3)`,
		userID, tokenHash, exp)
	if err != nil {
		return "", err
	}
	return raw, nil
}

func (r *Repo) ResetPasswordWithToken(ctx context.Context, rawToken, newPassword string) error {
	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])
	var userID string
	var used *time.Time
	err := r.Pool.QueryRow(ctx, `
SELECT user_id::text, used_at FROM password_reset_tokens
WHERE token_hash = $1 AND expires_at > now() ORDER BY created_at DESC LIMIT 1`, tokenHash).
		Scan(&userID, &used)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("invalid or expired token")
	}
	if err != nil {
		return err
	}
	if used != nil {
		return fmt.Errorf("token already used")
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2::uuid`, hash, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE password_reset_tokens SET used_at = now() WHERE token_hash = $1`, tokenHash); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
