package moviecategory

import (
	"errors"
	"strings"
	"time"
)

const MaxNameLength = 100

// Ошибки домена.
var (
	ErrProfileIDRequired = errors.New("profile id is required")
	ErrNameRequired      = errors.New("name is required")
	ErrNameTooLong       = errors.New("name must be 100 characters or fewer")
	ErrNotFound          = errors.New("movie category not found")
)

// MovieCategory — короткая персональная категория/тег фильма (напр. «Для вечера»,
// «Атмосферное»), принадлежащая конкретному профилю. Аналог streamingsite.StreamingSite,
// но без url/language — просто название.
type MovieCategory struct {
	id        int64
	profileID int64
	name      string
	position  int
	enabled   bool
	createdAt time.Time
	updatedAt time.Time
}

// NewMovieCategory создаёт новую валидную MovieCategory (без сохранения в БД). Включена по умолчанию.
func NewMovieCategory(profileID int64, name string, position int) (*MovieCategory, error) {
	if profileID <= 0 {
		return nil, ErrProfileIDRequired
	}

	c := &MovieCategory{profileID: profileID, position: position, enabled: true}
	if err := c.SetName(name); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	c.createdAt = now
	c.updatedAt = now
	return c, nil
}

// Reconstitute восстанавливает MovieCategory из БД без валидации.
func Reconstitute(id, profileID int64, name string, position int, enabled bool, createdAt, updatedAt time.Time) *MovieCategory {
	return &MovieCategory{
		id:        id,
		profileID: profileID,
		name:      name,
		position:  position,
		enabled:   enabled,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// ── Геттеры ──────────────────────────────────────────────────────────────────

func (c *MovieCategory) ID() int64            { return c.id }
func (c *MovieCategory) ProfileID() int64     { return c.profileID }
func (c *MovieCategory) Name() string         { return c.name }
func (c *MovieCategory) Position() int        { return c.position }
func (c *MovieCategory) Enabled() bool        { return c.enabled }
func (c *MovieCategory) CreatedAt() time.Time { return c.createdAt }
func (c *MovieCategory) UpdatedAt() time.Time { return c.updatedAt }

// ── Сеттеры ──────────────────────────────────────────────────────────────────

func (c *MovieCategory) SetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	if len([]rune(name)) > MaxNameLength {
		return ErrNameTooLong
	}
	c.name = name
	c.touch()
	return nil
}

func (c *MovieCategory) SetPosition(position int) {
	c.position = position
	c.touch()
}

// SetEnabled переключает видимость категории в дропдауне на фронте — отключённая
// категория остаётся в списке настроек, но не предлагается при добавлении фильма.
func (c *MovieCategory) SetEnabled(enabled bool) {
	c.enabled = enabled
	c.touch()
}

func (c *MovieCategory) touch() { c.updatedAt = time.Now().UTC() }
