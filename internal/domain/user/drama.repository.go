package user

import "context"

// DramaRepository — интерфейс персистентности для дорам пользователя.
type DramaRepository interface {
	// GetByUserID возвращает все дорамы пользователя вместе с тегами.
	GetByUserID(ctx context.Context, userID int64) ([]*Drama, error)
}

// BadgeRepository — интерфейс персистентности для бэйджей пользователя.
type BadgeRepository interface {
	// GetByUserID возвращает все бэйджи пользователя.
	GetByUserID(ctx context.Context, userID int64) ([]*Badge, error)
}
