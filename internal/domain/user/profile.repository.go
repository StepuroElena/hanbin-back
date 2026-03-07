package user

import "context"

// Repository — интерфейс персистентности для домена пользователя.
// Реализации живут в internal/repository/user.
// Принцип инверсии зависимостей: домен знает только об интерфейсе,
// конкретная БД его не касается.
type Repository interface {
	// Create сохраняет новый Profile и возвращает присвоенный ID.
	// Возвращает ErrEmailNotUnique, если email уже занят.
	Create(ctx context.Context, profile *Profile) (int64, error)

	// GetByID возвращает Profile по первичному ключу.
	// Возвращает ErrNotFound, если запись не найдена.
	GetByID(ctx context.Context, id int64) (*Profile, error)

	// GetByEmail возвращает Profile по email.
	// Возвращает ErrNotFound, если запись не найдена.
	GetByEmail(ctx context.Context, email string) (*Profile, error)

	// Update сохраняет изменённые поля Profile.
	// Возвращает ErrEmailNotUnique при коллизии email.
	Update(ctx context.Context, profile *Profile) error

	// Delete удаляет Profile по ID.
	Delete(ctx context.Context, id int64) error
}
