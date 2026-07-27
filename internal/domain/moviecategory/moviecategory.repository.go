package moviecategory

import "context"

// Repository — интерфейс персистентности для категорий фильмов.
type Repository interface {
	// Create сохраняет новую MovieCategory и возвращает присвоенный ID.
	Create(ctx context.Context, category *MovieCategory) (int64, error)

	// GetAllByProfileID возвращает все категории профиля, отсортированные по position, id.
	GetAllByProfileID(ctx context.Context, profileID int64) ([]*MovieCategory, error)

	// GetByID возвращает категорию по первичному ключу. Возвращает ErrNotFound, если не найдена.
	GetByID(ctx context.Context, id int64) (*MovieCategory, error)

	// Update сохраняет изменённые поля (name, position, enabled).
	Update(ctx context.Context, category *MovieCategory) error

	// Delete удаляет категорию по ID.
	Delete(ctx context.Context, id int64) error

	// CountByProfileID возвращает количество категорий у профиля — используется, чтобы понять,
	// нужно ли досеять дефолтные категории новому/старому пользователю.
	CountByProfileID(ctx context.Context, profileID int64) (int, error)
}
