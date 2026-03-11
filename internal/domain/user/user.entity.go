package user

import (
	"errors"
	"strings"
	"time"
)

// Ошибки, специфичные для агрегата User.
var (
	ErrUserNotFound   = errors.New("user not found")
	ErrUserEmailTaken = errors.New("email is already taken")
)

// User — агрегат учётной записи (auth).
// Хранит данные для идентификации: name + email + password_hash.
type User struct {
	id           int64
	name         string
	email        string
	passwordHash string
	createdAt    time.Time
	updatedAt    time.Time
}

// NewUser создаёт валидный User. Хэш пароля уже вычислен сервисом.
func NewUser(name, email, passwordHash string) (*User, error) {
	u := &User{}
	if err := u.SetName(name); err != nil {
		return nil, err
	}
	if err := u.SetEmail(email); err != nil {
		return nil, err
	}
	if passwordHash == "" {
		return nil, ErrPasswordRequired
	}
	u.passwordHash = passwordHash

	now := time.Now().UTC()
	u.createdAt = now
	u.updatedAt = now
	return u, nil
}

// ReconstituteUser восстанавливает User из БД без валидации.
func ReconstituteUser(id int64, name, email, passwordHash string, createdAt, updatedAt time.Time) *User {
	return &User{
		id:           id,
		name:         name,
		email:        email,
		passwordHash: passwordHash,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

// ── Геттеры ──────────────────────────────────────────────────────────────────

func (u *User) ID() int64            { return u.id }
func (u *User) Name() string         { return u.name }
func (u *User) Email() string        { return u.email }
func (u *User) PasswordHash() string { return u.passwordHash }
func (u *User) CreatedAt() time.Time { return u.createdAt }
func (u *User) UpdatedAt() time.Time { return u.updatedAt }

// ── Сеттеры ──────────────────────────────────────────────────────────────────

func (u *User) SetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	if len([]rune(name)) > MaxNameLength {
		return ErrNameTooLong
	}
	u.name = name
	u.updatedAt = time.Now().UTC()
	return nil
}

func (u *User) SetEmail(email string) error {
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
	u.email = email
	u.updatedAt = time.Now().UTC()
	return nil
}
