package user

import (
	"context"
	"fmt"
	"strings"

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
