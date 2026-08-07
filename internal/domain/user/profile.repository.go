package user

import "context"

// Repository — интерфейс персистентности для домена пользователя.
type Repository interface {
	// Create сохраняет новый Profile и возвращает присвоенный ID.
	// Возвращает ErrProfileExists, если профиль для данного user_id уже есть.
	Create(ctx context.Context, profile *Profile) (int64, error)

	// GetByID возвращает Profile по первичному ключу.
	// Возвращает ErrNotFound, если запись не найдена.
	GetByID(ctx context.Context, id int64) (*Profile, error)

// GetByEmail возвращает Profile по email (нужен для логина).
	// Возвращает ErrNotFound, если запись не найдена.
	GetByEmail(ctx context.Context, email string) (*Profile, error)

// Update сохраняет изменённые поля Profile (name, email).
	Update(ctx context.Context, profile *Profile) error

	// UpdatePassword обновляет password_hash для профиля по ID.
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error

	// ConfirmEmail проставляет email_confirmed_at = текущее время для профиля по ID.
	// Вызывается после успешной валидации токена подтверждения (см. auth.Service.ConfirmEmail).
	ConfirmEmail(ctx context.Context, id int64) error

	// Delete удаляет Profile по ID.
	Delete(ctx context.Context, id int64) error
}
