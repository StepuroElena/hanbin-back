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
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterOutput — ответ на успешную регистрацию.
// Возвращаем только user_id — клиент использует его для создания профиля.
type RegisterOutput struct {
	UserID int64 `json:"user_id"`
}

// Register создаёт учётную запись пользователя.
// Хэширует пароль, сохраняет в таблицу users, возвращает user_id.
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

	u, err := domain.NewUser(in.Email, string(hash))
	if err != nil {
		return nil, fmt.Errorf("service.Register: %w", err)
	}

	id, err := s.userRepo.Create(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("service.Register: %w", err)
	}

	return &RegisterOutput{UserID: id}, nil
}
