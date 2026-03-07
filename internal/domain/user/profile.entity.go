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
	ErrNameRequired   = errors.New("name is required")
	ErrEmailRequired  = errors.New("email is required")
	ErrNameTooLong    = errors.New("name must be 255 characters or fewer")
	ErrEmailTooLong   = errors.New("email must be 255 characters or fewer")
	ErrEmailInvalid   = errors.New("email format is invalid")
	ErrEmailNotUnique = errors.New("email is already taken")
	ErrNotFound       = errors.New("user not found")
)

// Profile — агрегат пользователя.
// Поля приватны, доступ только через конструктор и сеттеры,
// что гарантирует соблюдение инвариантов домена.
type Profile struct {
	id        int64
	name      string
	email     string
	createdAt time.Time
	updatedAt time.Time
}

// NewProfile создаёт валидный Profile. Не сохраняет ничего в БД —
// за персистентность отвечает Repository.
func NewProfile(name, email string) (*Profile, error) {
	p := &Profile{}

	if err := p.SetName(name); err != nil {
		return nil, err
	}
	if err := p.SetEmail(email); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	p.createdAt = now
	p.updatedAt = now

	return p, nil
}

// Reconstitute восстанавливает Profile из персистентного хранилища.
// Валидация пропускается — данные уже были проверены при сохранении.
func Reconstitute(id int64, name, email string, createdAt, updatedAt time.Time) *Profile {
	return &Profile{
		id:        id,
		name:      name,
		email:     email,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// ── Геттеры ──────────────────────────────────────────────────────────────────

func (p *Profile) ID() int64          { return p.id }
func (p *Profile) Name() string       { return p.name }
func (p *Profile) Email() string      { return p.email }
func (p *Profile) CreatedAt() time.Time { return p.createdAt }
func (p *Profile) UpdatedAt() time.Time { return p.updatedAt }

// ── Сеттеры с валидацией ─────────────────────────────────────────────────────

// SetName обновляет имя пользователя с проверкой бизнес-правил.
// Имя — произвольная строка, обязательная, не длиннее 255 символов.
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

// SetEmail обновляет email с проверкой бизнес-правил.
// Уникальность НЕ проверяется здесь — это зона ответственности Repository.
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

// isValidEmail — лёгкая проверка формата email без внешних зависимостей.
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
