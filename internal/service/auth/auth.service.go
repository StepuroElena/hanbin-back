package auth

import (
	"context"
	"fmt"
	"strings"

	authdomain "github.com/hanbin/hanbin-back/internal/domain/auth"
	userdomain "github.com/hanbin/hanbin-back/internal/domain/user"
	"github.com/hanbin/hanbin-back/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

// Service реализует регистрацию, логин и смену пароля.
type Service struct {
	repo userdomain.Repository
}

func NewService(repo userdomain.Repository) *Service {
	return &Service{repo: repo}
}

// ── DTO ───────────────────────────────────────────────────────────────────────

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterOutput struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginOutput struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	Token  string `json:"token"`
}

type SetPasswordInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ── Use cases ─────────────────────────────────────────────────────────────────

// Register создаёт нового пользователя с хешированным паролем.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*RegisterOutput, error) {
	password := strings.TrimSpace(in.Password)
	if password == "" {
		return nil, authdomain.ErrPasswordRequired
	}
	if len([]rune(password)) < userdomain.MinPasswordLength {
		return nil, authdomain.ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("auth.Register: bcrypt: %w", err)
	}

	profile, err := userdomain.NewProfile(in.Name, in.Email, string(hash))
	if err != nil {
		return nil, fmt.Errorf("auth.Register: %w", err)
	}

	id, err := s.repo.Create(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("auth.Register: %w", err)
	}

	return &RegisterOutput{
		UserID: id,
		Name:   profile.Name(),
		Email:  profile.Email(),
	}, nil
}

// Login проверяет credentials и возвращает JWT-токен.
func (s *Service) Login(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	if strings.TrimSpace(in.Email) == "" || strings.TrimSpace(in.Password) == "" {
		return nil, authdomain.ErrPasswordRequired
	}

	profile, err := s.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil {
		return nil, authdomain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash()), []byte(in.Password)); err != nil {
		return nil, authdomain.ErrInvalidCredentials
	}

	token, err := middleware.IssueJWT(profile.ID())
	if err != nil {
		return nil, fmt.Errorf("auth.Login: issue token: %w", err)
	}

	return &LoginOutput{
		UserID: profile.ID(),
		Email:  profile.Email(),
		Token:  token,
	}, nil
}

// SetPassword устанавливает новый пароль для существующего пользователя по email.
// Используется для исправления пустого password_hash у старых пользователей.
func (s *Service) SetPassword(ctx context.Context, in SetPasswordInput) error {
	password := strings.TrimSpace(in.Password)
	if password == "" {
		return authdomain.ErrPasswordRequired
	}
	if len([]rune(password)) < userdomain.MinPasswordLength {
		return authdomain.ErrPasswordTooShort
	}

	profile, err := s.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(in.Email)))
	if err != nil {
		return userdomain.ErrNotFound
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth.SetPassword: bcrypt: %w", err)
	}

	return s.repo.UpdatePassword(ctx, profile.ID(), string(hash))
}
