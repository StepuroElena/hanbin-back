package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/auth"
)

type postgresConfirmationTokenRepository struct {
	db *sql.DB
}

// NewPostgresConfirmationTokenRepository создаёт репозиторий токенов подтверждения email для PostgreSQL.
func NewPostgresConfirmationTokenRepository(db *sql.DB) domain.ConfirmationTokenRepository {
	return &postgresConfirmationTokenRepository{db: db}
}

func (r *postgresConfirmationTokenRepository) Create(ctx context.Context, profileID int64, token string, expiresAt time.Time) error {
	const q = `INSERT INTO email_confirmation_tokens (profile_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, q, profileID, token, expiresAt, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("confirmation_token repository.Create: %w", err)
	}
	return nil
}

func (r *postgresConfirmationTokenRepository) GetByToken(ctx context.Context, token string) (*domain.ConfirmationToken, error) {
	const q = `SELECT profile_id, token, expires_at, used_at FROM email_confirmation_tokens WHERE token = $1`

	var (
		profileID int64
		tok       string
		expiresAt time.Time
		usedAt    sql.NullTime
	)
	err := r.db.QueryRowContext(ctx, q, token).Scan(&profileID, &tok, &expiresAt, &usedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrConfirmationTokenInvalid
		}
		return nil, fmt.Errorf("confirmation_token repository.GetByToken: %w", err)
	}

	ct := &domain.ConfirmationToken{ProfileID: profileID, Token: tok, ExpiresAt: expiresAt}
	if usedAt.Valid {
		t := usedAt.Time
		ct.UsedAt = &t
	}
	return ct, nil
}

func (r *postgresConfirmationTokenRepository) MarkUsed(ctx context.Context, token string) error {
	const q = `UPDATE email_confirmation_tokens SET used_at = $1 WHERE token = $2`
	_, err := r.db.ExecContext(ctx, q, time.Now().UTC(), token)
	if err != nil {
		return fmt.Errorf("confirmation_token repository.MarkUsed: %w", err)
	}
	return nil
}
