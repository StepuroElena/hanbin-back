package auth

import (
	"context"
	"time"
)

// ConfirmationToken — одноразовый токен подтверждения email при регистрации, живёт
// ограниченное время (24 часа, см. service.Register). Та же форма, что и ResetToken —
// сознательно не переиспользуем одну таблицу/тип для обоих сценариев, чтобы токен
// восстановления пароля нельзя было случайно использовать для подтверждения почты и наоборот.
type ConfirmationToken struct {
	ProfileID int64
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time // nil = ещё не использован
}

// ConfirmationTokenRepository — персистентность токенов подтверждения email.
type ConfirmationTokenRepository interface {
	// Create сохраняет новый токен для профиля с указанным сроком действия.
	Create(ctx context.Context, profileID int64, token string, expiresAt time.Time) error

	// GetByToken возвращает токен по значению. ErrConfirmationTokenInvalid, если такого токена нет.
	GetByToken(ctx context.Context, token string) (*ConfirmationToken, error)

	// MarkUsed помечает токен использованным — повторно применить его будет нельзя.
	MarkUsed(ctx context.Context, token string) error
}
