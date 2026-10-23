package voiceover

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

// MaxNameLength совпадает с dramas.voiceover VARCHAR(255) — название озвучки пишется в дораму как есть.
const MaxNameLength = 255

// Ошибки домена.
var (
	ErrProfileIDRequired = errors.New("profile id is required")
	ErrNameRequired      = errors.New("name is required")
	ErrNameTooLong       = errors.New("name must be 255 characters or fewer")
	ErrNotFound          = errors.New("voiceover not found")
	ErrDuplicateName     = errors.New("voiceover with this name already exists")
	ErrDefaultReadOnly   = errors.New("default voiceover can only be enabled or disabled")
)

// Voiceover — озвучка (студия перевода). Бывает двух видов:
//   - дефолтная (profileID == 0) — общая для всех, её можно только выключить/включить у себя;
//   - собственная (profileID > 0) — принадлежит профилю, её можно переименовать и удалить.
//
// enabled — персональное состояние тогла для профиля, от лица которого прочитана озвучка.
type Voiceover struct {
	id        int64
	profileID int64 // 0 = дефолтная
	name      string
	position  int
	enabled   bool
	createdAt time.Time
	updatedAt time.Time
}

// NewCustom создаёт новую собственную озвучку профиля (без сохранения в БД). Включена по умолчанию.
func NewCustom(profileID int64, name string, position int) (*Voiceover, error) {
	if profileID <= 0 {
		return nil, ErrProfileIDRequired
	}
	v := &Voiceover{profileID: profileID, position: position, enabled: true}
	if err := v.SetName(name); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	v.createdAt = now
	v.updatedAt = now
	return v, nil
}

// Reconstitute восстанавливает Voiceover из БД без валидации. profileID == 0 — дефолтная.
func Reconstitute(id, profileID int64, name string, position int, enabled bool, createdAt, updatedAt time.Time) *Voiceover {
	return &Voiceover{
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

func (v *Voiceover) ID() int64            { return v.id }
func (v *Voiceover) ProfileID() int64     { return v.profileID }
func (v *Voiceover) Name() string         { return v.name }
func (v *Voiceover) Position() int        { return v.position }
func (v *Voiceover) Enabled() bool        { return v.enabled }
func (v *Voiceover) CreatedAt() time.Time { return v.createdAt }
func (v *Voiceover) UpdatedAt() time.Time { return v.updatedAt }

// IsDefault — общая для всех озвучка (не принадлежит ни одному профилю).
func (v *Voiceover) IsDefault() bool { return v.profileID == 0 }

// ── Сеттеры ──────────────────────────────────────────────────────────────────

// SetName переименовывает озвучку. Дефолтные переименовывать нельзя.
func (v *Voiceover) SetName(name string) error {
	if v.id != 0 && v.IsDefault() {
		return ErrDefaultReadOnly
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	if len([]rune(name)) > MaxNameLength {
		return ErrNameTooLong
	}
	v.name = name
	v.touch()
	return nil
}

func (v *Voiceover) SetEnabled(enabled bool) {
	v.enabled = enabled
	v.touch()
}

func (v *Voiceover) touch() { v.updatedAt = time.Now().UTC() }

// NormalizeName приводит название к форме для сравнения: нижний регистр, только буквы и цифры.
// "DubLikTV" и "DubLik.TV" дают одно и то же — так сравниваем названия со скрейпа и ищем дубли.
func NormalizeName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
