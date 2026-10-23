package voiceover

import "context"

// Repository — интерфейс персистентности для озвучек.
type Repository interface {
	// ListForProfile возвращает дефолтные озвучки + собственные озвучки профиля
	// с персональным состоянием тогла. Сортировка: сначала дефолтные, потом свои; внутри — position, id.
	ListForProfile(ctx context.Context, profileID int64) ([]*Voiceover, error)

	// ListDefaults возвращает только дефолтные озвучки (все включены) — для гостя без аккаунта.
	ListDefaults(ctx context.Context) ([]*Voiceover, error)

	// GetForProfile возвращает озвучку, доступную профилю (дефолтную или его собственную),
	// с персональным состоянием тогла. Чужая или несуществующая — ErrNotFound.
	GetForProfile(ctx context.Context, profileID, id int64) (*Voiceover, error)

	// NextCustomPosition — позиция для новой собственной озвучки (в конец списка профиля).
	NextCustomPosition(ctx context.Context, profileID int64) (int, error)

	// Create сохраняет новую собственную озвучку и возвращает присвоенный ID.
	Create(ctx context.Context, v *Voiceover) (int64, error)

	// Rename переименовывает собственную озвучку профиля и в той же транзакции
	// обновляет поле voiceover у дорам профиля, где стояло старое название.
	Rename(ctx context.Context, profileID, id int64, oldName, newName string) error

	// SetEnabled сохраняет персональный тогл профиля для озвучки (upsert).
	SetEnabled(ctx context.Context, profileID, id int64, enabled bool) error

	// DeleteCustom удаляет собственную озвучку профиля.
	DeleteCustom(ctx context.Context, profileID, id int64) error
}
