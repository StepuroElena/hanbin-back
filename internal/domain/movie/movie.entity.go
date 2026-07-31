package movie

import (
	"errors"
	"strings"
	"time"
)

// ── Константы и ошибки домена ─────────────────────────────────────────────────

const (
	MaxTitleLength   = 500
	MaxGenreLength   = 100
	MaxCountryLength = 100
	MinYear          = 1900
	MaxYear          = 2100
)

var (
	ErrTitleRequired      = errors.New("title is required")
	ErrTitleTooLong       = errors.New("title must be 500 characters or fewer")
	ErrGenreRequired      = errors.New("genre is required")
	ErrGenreTooLong       = errors.New("genre must be 100 characters or fewer")
	ErrCountryTooLong     = errors.New("country must be 100 characters or fewer")
	ErrCategoryTooLong    = errors.New("category must be 100 characters or fewer")
	ErrInvalidYear        = errors.New("release_year must be between 1900 and 2100")
	ErrProfileIDRequired  = errors.New("profile_id is required")
	ErrNotFound           = errors.New("movie not found")
	ErrNotArchived        = errors.New("movie must be archived before deletion")
	ErrInvalidWatchStatus = errors.New("watch_status must be one of 'planned', 'watching', 'completed', 'dropped'")
)

// ── Enum-тип ──────────────────────────────────────────────────────────────────

// WatchStatus — статус просмотра фильма. Теперь четыре значения — так же, как и у дорам
// (нужно для фильтр-чипсов «Все/Смотрю/Просмотрено/Запланировано/Брошено» на фронте).
type WatchStatus string

const (
	WatchStatusPlanned   WatchStatus = "planned"
	WatchStatusWatching  WatchStatus = "watching"
	WatchStatusCompleted WatchStatus = "completed"
	WatchStatusDropped   WatchStatus = "dropped"
)

func ParseWatchStatus(s string) (WatchStatus, error) {
	switch WatchStatus(s) {
	case WatchStatusPlanned, WatchStatusWatching, WatchStatusCompleted, WatchStatusDropped:
		return WatchStatus(s), nil
	}
	return "", ErrInvalidWatchStatus
}

// ── Агрегат ───────────────────────────────────────────────────────────────────

// Movie — агрегат фильма. Минимальная версия: название + жанр (оба обязательны) +
// опциональный год выпуска + статус просмотра + архив (так же, как у дорам).
type Movie struct {
	id          int64
	profileID   int64
	title       string
	genre       string
	country     string // "" = не указана, опциональное поле
	category    string // "" = не указана, опциональное поле — значение из персонального списка категорий
	releaseYear *int // nil = не указан
	watchStatus WatchStatus
	isArchived  bool
	createdAt   time.Time
	updatedAt   time.Time
}

// NewMovie создаёт валидный агрегат Movie. Статус при создании всегда "planned", архив — false,
// так же, как у дорам при добавлении.
func NewMovie(profileID int64, title, genre, country, category string, releaseYear *int) (*Movie, error) {
	if profileID <= 0 {
		return nil, ErrProfileIDRequired
	}

	m := &Movie{profileID: profileID, watchStatus: WatchStatusPlanned, isArchived: false}

	if err := m.setTitle(title); err != nil {
		return nil, err
	}
	if err := m.setGenre(genre); err != nil {
		return nil, err
	}
	if err := m.setCountry(country); err != nil {
		return nil, err
	}
	if err := m.setCategory(category); err != nil {
		return nil, err
	}

	if releaseYear != nil {
		if *releaseYear < MinYear || *releaseYear > MaxYear {
			return nil, ErrInvalidYear
		}
		v := *releaseYear
		m.releaseYear = &v
	}

	now := time.Now().UTC()
	m.createdAt = now
	m.updatedAt = now

	return m, nil
}

// Reconstitute восстанавливает Movie из БД без валидации.
func Reconstitute(id, profileID int64, title, genre, country, category string, releaseYear *int, watchStatus WatchStatus, isArchived bool, createdAt, updatedAt time.Time) *Movie {
	return &Movie{
		id:          id,
		profileID:   profileID,
		title:       title,
		genre:       genre,
		country:     country,
		category:    category,
		releaseYear: releaseYear,
		watchStatus: watchStatus,
		isArchived:  isArchived,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// ── Геттеры ───────────────────────────────────────────────────────────────────

func (m *Movie) ID() int64                { return m.id }
func (m *Movie) ProfileID() int64         { return m.profileID }
func (m *Movie) Title() string            { return m.title }
func (m *Movie) Genre() string            { return m.genre }
func (m *Movie) Country() string          { return m.country }
func (m *Movie) Category() string         { return m.category }
func (m *Movie) ReleaseYear() *int        { return m.releaseYear }
func (m *Movie) WatchStatus() WatchStatus { return m.watchStatus }
func (m *Movie) IsArchived() bool         { return m.isArchived }
func (m *Movie) CreatedAt() time.Time     { return m.createdAt }
func (m *Movie) UpdatedAt() time.Time     { return m.updatedAt }

// ── Приватные сеттеры ─────────────────────────────────────────────────────────

func (m *Movie) setTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrTitleRequired
	}
	if len([]rune(title)) > MaxTitleLength {
		return ErrTitleTooLong
	}
	m.title = title
	return nil
}

func (m *Movie) setGenre(genre string) error {
	genre = strings.TrimSpace(genre)
	if genre == "" {
		return ErrGenreRequired
	}
	if len([]rune(genre)) > MaxGenreLength {
		return ErrGenreTooLong
	}
	m.genre = genre
	return nil
}

// setCountry устанавливает страну выпуска. Поле опциональное — пустая строка допустима.
func (m *Movie) setCountry(country string) error {
	country = strings.TrimSpace(country)
	if len([]rune(country)) > MaxCountryLength {
		return ErrCountryTooLong
	}
	m.country = country
	return nil
}

// setCategory устанавливает категорию фильма (значение из персонального списка movie_categories).
// Поле опциональное — пустая строка допустима.
func (m *Movie) setCategory(category string) error {
	category = strings.TrimSpace(category)
	if len([]rune(category)) > MaxCountryLength {
		return ErrCategoryTooLong
	}
	m.category = category
	return nil
}

// ── Экспортируемые сеттеры (для частичного обновления через PATCH /movies/{id}) ──
// Та же валидация, что и при создании — просто публичные обёртки над приватными setX.

func (m *Movie) SetTitle(title string) error    { return m.setTitle(title) }
func (m *Movie) SetGenre(genre string) error    { return m.setGenre(genre) }
func (m *Movie) SetCountry(country string) error { return m.setCountry(country) }
func (m *Movie) SetCategory(category string) error { return m.setCategory(category) }

// SetReleaseYear обновляет год выпуска. nil сбрасывает поле (год не указан).
func (m *Movie) SetReleaseYear(year *int) error {
	if year == nil {
		m.releaseYear = nil
		return nil
	}
	if *year < MinYear || *year > MaxYear {
		return ErrInvalidYear
	}
	v := *year
	m.releaseYear = &v
	return nil
}

// SetWatchStatus обновляет статус просмотра — вызывающий код уже провалидировал строку через ParseWatchStatus.
func (m *Movie) SetWatchStatus(status WatchStatus) {
	m.watchStatus = status
}
