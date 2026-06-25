package drama

import "context"

// Repository — интерфейс персистентности для домена драм.
type Repository interface {
	// Create сохраняет новую Drama и возвращает присвоенный ID.
	Create(ctx context.Context, d *Drama) (int64, error)

	// GetAllByProfileID возвращает все дорамы пользователя.
	GetAllByProfileID(ctx context.Context, profileID int64) ([]*Drama, error)
}
