package user

import (
	"errors"
	"strings"
	"time"
)

const (
	MaxNameLength  = 255
	MaxEmailLength = 255
)

// Ошибки домена — используются во всех слоях приложения.
var (
	ErrNameRequired     = errors.New("name is required")
	ErrEmailRequired    = errors.New("email is required")
	ErrPasswordRequired = errors.New("password is required")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrNameTooLong      = errors.New("name must be 255 characters or fewer")
	ErrEmailTooLong     = errors.New("email must be 255 characters or fewer")
	ErrEmailInvalid     = errors.New("email format is invalid")
	ErrEmailNotUnique   = errors.New("email is already taken")
	ErrNotFound         = errors.New("profile not found")
	ErrUserIDRequired   = errors.New("user_id is required")
	ErrProfileExists    = errors.New("profile already exists for this user")
)

// Profile — агрегат публичного профиля пользователя.
// Привязан к User через userID (FK → users.id).
// Не хранит email и пароль — это зона ответственности User.
type Profile struct {
	id        int64
	userID    int64
	name      string
	createdAt time.Time
	updatedAt time.Time
}

// NewProfile создаёт валидный Profile для существующего пользователя.
func NewProfile(userID int64, name string) (*Profile, error) {
	if userID <= 0 {
		return nil, ErrUserIDRequired
	}
	p := &Profile{userID: userID}
	if err := p.SetName(name); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	p.createdAt = now
	p.updatedAt = now
	return p, nil
}

// Reconstitute восстанавливает Profile из БД без валидации.
func Reconstitute(id, userID int64, name string, createdAt, updatedAt time.Time) *Profile {
	return &Profile{
		id:        id,
		userID:    userID,
		name:      name,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// ── Геттеры ──────────────────────────────────────────────────────────────────

func (p *Profile) ID() int64          { return p.id }
func (p *Profile) UserID() int64      { return p.userID }
func (p *Profile) Name() string       { return p.name }
func (p *Profile) CreatedAt() time.Time { return p.createdAt }
func (p *Profile) UpdatedAt() time.Time { return p.updatedAt }

// ── Сеттеры с валидацией ─────────────────────────────────────────────────────

func (p *Profile) SetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	if len([]rune(name)) > MaxNameLength {
		return ErrNameTooLong
	}
	p.name = name
	p.updatedAt = time.Now().UTC()
	return nil
}

// ── Вспомогательные функции ───────────────────────────────────────────────────

func isValidEmail(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 1 {
		return false
	}
	local := email[:at]
	domain := email[at+1:]
	if len(local) == 0 || len(domain) < 3 {
		return false
	}
	return strings.Contains(domain, ".")
}
