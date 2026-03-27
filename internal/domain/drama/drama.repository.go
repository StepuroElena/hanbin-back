package drama

import "context"

// Repository — интерфейс персистентности для домена драм.
type Repository interface {
	// Create сохраняет новую Drama и возвращает присвоенный ID.
	Create(ctx context.Context, d *Drama) (int64, error)

	// GetAllByProfileID возвращает все дорамы пользователя.
	GetAllByProfileID(ctx context.Context, profileID int64) ([]*Drama, error)

	// GetByID возвращает дораму по ID.
	GetByID(ctx context.Context, id int64) (*Drama, error)

	// UpdateArchived обновляет флаг is_archived у дорамы.
	UpdateArchived(ctx context.Context, id int64, isArchived bool) error

	// Delete удаляет дораму из БД по ID.
	Delete(ctx context.Context, id int64) error
}
