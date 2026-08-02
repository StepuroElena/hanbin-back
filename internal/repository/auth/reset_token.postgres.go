package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/auth"
)

type postgresResetTokenRepository struct {
	db *sql.DB
}

// NewPostgresResetTokenRepository создаёт репозиторий токенов восстановления пароля для PostgreSQL.
func NewPostgresResetTokenRepository(db *sql.DB) domain.ResetTokenRepository {
	return &postgresResetTokenRepository{db: db}
}

func (r *postgresResetTokenRepository) Create(ctx context.Context, profileID int64, token string, expiresAt time.Time) error {
	const q = `INSERT INTO password_reset_tokens (profile_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, q, profileID, token, expiresAt, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("reset_token repository.Create: %w", err)
	}
	return nil
}

func (r *postgresResetTokenRepository) GetByToken(ctx context.Context, token string) (*domain.ResetToken, error) {
	const q = `SELECT profile_id, token, expires_at, used_at FROM password_reset_tokens WHERE token = $1`

	var (
		profileID int64
		tok       string
		expiresAt time.Time
		usedAt    sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, q, token).Scan(&profileID, &tok, &expiresAt, &usedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTokenInvalid
		}
		return nil, fmt.Errorf("reset_token repository.GetByToken: %w", err)
	}

	rt := &domain.ResetToken{ProfileID: profileID, Token: tok, ExpiresAt: expiresAt}
	if usedAt.Valid {
		t := usedAt.Time
		rt.UsedAt = &t
	}
	return rt, nil
}

func (r *postgresResetTokenRepository) MarkUsed(ctx context.Context, token string) error {
	const q = `UPDATE password_reset_tokens SET used_at = $1 WHERE token = $2`
	_, err := r.db.ExecContext(ctx, q, time.Now().UTC(), token)
	if err != nil {
		return fmt.Errorf("reset_token repository.MarkUsed: %w", err)
	}
	return nil
}
