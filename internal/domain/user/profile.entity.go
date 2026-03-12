package user

import (
	"errors"
	"strings"
	"time"
)

const (
	MaxNameLength     = 255
	MaxEmailLength    = 255
	MinPasswordLength = 6
)

// Ошибки домена — используются во всех слоях приложения.
var (
	ErrNameRequired     = errors.New("name is required")
	ErrEmailRequired    = errors.New("email is required")
	ErrPasswordRequired = errors.New("password is required")
	ErrPasswordTooShort = errors.New("password must be at least 6 characters")
	ErrNameTooLong      = errors.New("name must be 255 characters or fewer")
	ErrEmailTooLong     = errors.New("email must be 255 characters or fewer")
	ErrEmailInvalid     = errors.New("email format is invalid")
	ErrEmailNotUnique   = errors.New("email is already taken")
	ErrNotFound         = errors.New("user not found")
)

// Profile — агрегат пользователя.
type Profile struct {
	id           int64
	name         string
	email        string
	passwordHash string
	createdAt    time.Time
	updatedAt    time.Time
}

// NewProfile создаёт валидный Profile без сохранения в БД.
// passwordHash — уже захешированный пароль (bcrypt), передаётся из сервиса.
func NewProfile(name, email, passwordHash string) (*Profile, error) {
	p := &Profile{}

	if err := p.SetName(name); err != nil {
		return nil, err
	}
	if err := p.SetEmail(email); err != nil {
		return nil, err
	}
	if strings.TrimSpace(passwordHash) == "" {
		return nil, ErrPasswordRequired
	}
	p.passwordHash = passwordHash

	now := time.Now().UTC()
	p.createdAt = now
	p.updatedAt = now

	return p, nil
}

// Reconstitute восстанавливает Profile из БД без валидации.
func Reconstitute(id int64, name, email, passwordHash string, createdAt, updatedAt time.Time) *Profile {
	return &Profile{
		id:           id,
		name:         name,
		email:        email,
		passwordHash: passwordHash,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

// ── Геттеры ──────────────────────────────────────────────────────────────────

func (p *Profile) ID() int64            { return p.id }
func (p *Profile) Name() string         { return p.name }
func (p *Profile) Email() string        { return p.email }
func (p *Profile) PasswordHash() string { return p.passwordHash }
func (p *Profile) CreatedAt() time.Time { return p.createdAt }
func (p *Profile) UpdatedAt() time.Time { return p.updatedAt }

// ── Сеттеры ───────────────────────────────────────────────────────────────────

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

func (p *Profile) SetEmail(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return ErrEmailRequired
	}
	if len(email) > MaxEmailLength {
		return ErrEmailTooLong
	}
	if !isValidEmail(email) {
		return ErrEmailInvalid
	}
	p.email = email
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
