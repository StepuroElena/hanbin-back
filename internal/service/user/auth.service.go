package user

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	domain "github.com/hanbin/hanbin-back/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	bcryptCost        = 12
)

// RegisterInput — входные данные для регистрации.
type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterOutput — ответ на успешную регистрацию.
type RegisterOutput struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

// jwtSecret возвращает секрет для подписи JWT.
// В продакшне берётся из переменной окружения JWT_SECRET.
func jwtSecret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("hanbin-dev-secret-change-in-prod")
}

// LoginInput — входные данные для входа.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginOutput — ответ на успешный вход.
type LoginOutput struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Token  string `json:"token"`
}

// Login проверяет учётные данные и возвращает JWT-токен.
// Оба поля обязательны. При несовпадении — ErrInvalidCredentials.
func (s *Service) Login(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" {
		return nil, fmt.Errorf("service.Login: %w", domain.ErrEmailRequired)
	}
	if strings.TrimSpace(in.Password) == "" {
		return nil, fmt.Errorf("service.Login: %w", domain.ErrPasswordRequired)
	}

	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Пользователь не найден — отдаём общую ошибку, чтобы не раскрывать наличие email.
		return nil, fmt.Errorf("service.Login: %w", domain.ErrInvalidCredentials)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash()), []byte(in.Password)); err != nil {
		return nil, fmt.Errorf("service.Login: %w", domain.ErrInvalidCredentials)
	}

	// Генерируем JWT: subject = user_id, срок действия 72 часа.
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", u.ID()),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(72 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtSecret())
	if err != nil {
		return nil, fmt.Errorf("service.Login: sign token: %w", err)
	}

	return &LoginOutput{
		UserID: u.ID(),
		Email:  u.Email(),
		Token:  signed,
	}, nil
}

// Register создаёт учётную запись пользователя.
// Валидирует пароль, хэширует через bcrypt, сохраняет в таблицу users.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*RegisterOutput, error) {
	password := strings.TrimSpace(in.Password)
	if password == "" {
		return nil, fmt.Errorf("service.Register: %w", domain.ErrPasswordRequired)
	}
	if len([]rune(password)) < minPasswordLength {
		return nil, fmt.Errorf("service.Register: %w", domain.ErrPasswordTooShort)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("service.Register: hash password: %w", err)
	}

	u, err := domain.NewUser(in.Name, in.Email, string(hash))
	if err != nil {
		return nil, fmt.Errorf("service.Register: %w", err)
	}

	id, err := s.userRepo.Create(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("service.Register: %w", err)
	}

	return &RegisterOutput{
		UserID: id,
		Name:   u.Name(),
		Email:  u.Email(),
	}, nil
}
