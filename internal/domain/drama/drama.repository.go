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

	// Update обновляет все редактируемые поля дорамы.
	Update(ctx context.Context, d *Drama) error

	// GetStatsByProfileID возвращает агрегированную статистику по статусам просмотра
	// для дорам пользователя (не архивированных), одним SQL-запросом.
	GetStatsByProfileID(ctx context.Context, profileID int64) (*Stats, error)

	// GetFacetsByProfileID возвращает список реально используемых стран и жанров в дорамах
	// пользователя (не архивированных) — для фильтров на главной: чтобы не показывать
	// чипы стран/жанров, которых ни у одной дорамы нет.
	GetFacetsByProfileID(ctx context.Context, profileID int64) (*Facets, error)
}
