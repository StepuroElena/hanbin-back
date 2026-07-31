package movie

import "context"

// Repository — интерфейс персистентности для домена фильмов.
type Repository interface {
	// Create сохраняет новый Movie и возвращает присвоенный ID.
	Create(ctx context.Context, m *Movie) (int64, error)

	// GetAllByProfileID возвращает все фильмы пользователя.
	GetAllByProfileID(ctx context.Context, profileID int64) ([]*Movie, error)

	// GetByID возвращает фильм по ID.
	GetByID(ctx context.Context, id int64) (*Movie, error)

	// UpdateWatchStatus обновляет статус просмотра фильма.
	UpdateWatchStatus(ctx context.Context, id int64, status WatchStatus) error

	// Update применяет частичное обновление фильма (название/жанр/страна/категория/год/статус) —
	// принимает уже изменённый агрегат, пишет все редактируемые поля одним UPDATE.
	Update(ctx context.Context, m *Movie) error

	// UpdateArchived обновляет флаг is_archived у фильма.
	UpdateArchived(ctx context.Context, id int64, isArchived bool) error

	// Delete удаляет фильм из БД по ID.
	Delete(ctx context.Context, id int64) error
}
