package user

import "context"

// UserRepository — интерфейс персистентности для агрегата User (auth).
type UserRepository interface {
	// Create сохраняет нового User и возвращает присвоенный ID.
	// Возвращает ErrUserEmailTaken, если email уже занят.
	Create(ctx context.Context, u *User) (int64, error)

	// GetByID возвращает User по первичному ключу.
	// Возвращает ErrUserNotFound, если запись не найдена.
	GetByID(ctx context.Context, id int64) (*User, error)

	// GetByEmail возвращает User по email.
	// Возвращает ErrUserNotFound, если запись не найдена.
	GetByEmail(ctx context.Context, email string) (*User, error)
}
