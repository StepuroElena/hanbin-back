package auth

import (
	"context"
	"time"
)

// ResetToken — одноразовый токен восстановления пароля, живёт ограниченное время (1 час, см. service.ForgotPassword).
type ResetToken struct {
	ProfileID int64
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time // nil = ещё не использован
}

// ResetTokenRepository — персистентность токенов восстановления пароля.
type ResetTokenRepository interface {
	// Create сохраняет новый токен для профиля с указанным сроком действия.
	Create(ctx context.Context, profileID int64, token string, expiresAt time.Time) error

	// GetByToken возвращает токен по значению. ErrTokenInvalid, если такого токена нет.
	GetByToken(ctx context.Context, token string) (*ResetToken, error)

	// MarkUsed помечает токен использованным — повторно применить его будет нельзя.
	MarkUsed(ctx context.Context, token string) error
}
