package streamingsite

import "context"

// Repository — интерфейс персистентности для сайтов просмотра.
type Repository interface {
	// Create сохраняет новый StreamingSite и возвращает присвоенный ID.
	Create(ctx context.Context, site *StreamingSite) (int64, error)

	// GetAllByProfileID возвращает все сайты профиля, отсортированные по position, id.
	GetAllByProfileID(ctx context.Context, profileID int64) ([]*StreamingSite, error)

	// GetByID возвращает сайт по первичному ключу. Возвращает ErrNotFound, если не найден.
	GetByID(ctx context.Context, id int64) (*StreamingSite, error)

	// Update сохраняет изменённые поля (name, url, language, position).
	Update(ctx context.Context, site *StreamingSite) error

	// Delete удаляет сайт по ID.
	Delete(ctx context.Context, id int64) error

	// CountByProfileID возвращает количество сайтов у профиля — используется, чтобы понять,
	// нужно ли досеять дефолтные сайты новому/старому пользователю.
	CountByProfileID(ctx context.Context, profileID int64) (int, error)
}
