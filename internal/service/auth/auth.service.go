package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	authdomain "github.com/hanbin/hanbin-back/internal/domain/auth"
	userdomain "github.com/hanbin/hanbin-back/internal/domain/user"
	"github.com/hanbin/hanbin-back/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

// Service реализует регистрацию, логин и смену пароля.
type Service struct {
	repo           userdomain.Repository
	resetTokenRepo authdomain.ResetTokenRepository
}

func NewService(repo userdomain.Repository, resetTokenRepo authdomain.ResetTokenRepository) *Service {
	return &Service{repo: repo, resetTokenRepo: resetTokenRepo}
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

// ForgotPasswordInput — тело запроса «Забыли пароль?».
type ForgotPasswordInput struct {
	Email string `json:"email"`
}

// ForgotPasswordOutput — ответ на запрос восстановления.
//
// ВРЕМЕННО (пока не подключён реальный email-провайдер, см. заметку в ForgotPassword): ResetLink
// отдаётся прямо в API-ответе, чтобы фронт мог показать его в UI вместо письма. Как только появится
// email-сервис — убрать это поле из ответа и реально отправлять письмо на Email пользователя,
// не раскрывая ссылку через API.
type ForgotPasswordOutput struct {
	ResetLink string `json:"reset_link"`
	ExpiresAt string `json:"expires_at"`
}

// ResetPasswordInput — тело запроса на установку нового пароля по токену из письма/ссылки.
type ResetPasswordInput struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

const passwordResetTTL = 1 * time.Hour

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

// ForgotPassword проверяет, что email существует, генерирует одноразовый токен восстановления
// (живёт passwordResetTTL) и возвращает ссылку на страницу смены пароля.
//
// Если email не найден — возвращает userdomain.ErrNotFound (404 на уровне хендлера), как явно
// просил продукт: пользователь должен видеть ошибку, если такой почты нет. Это раскрывает факт
// существования аккаунта по email (user enumeration) — сознательный компромисс по запросу продукта,
// не техническое упущение.
//
// ВРЕМЕННО: пока не подключён реальный email-провайдер, письмо не отправляется — вместо этого
// ссылка логируется на сервере и возвращается прямо в ответе (см. ForgotPasswordOutput). Как только
// появится email-сервис (Resend/SendGrid/SMTP) — убрать ResetLink из ответа и отправлять его письмом.
func (s *Service) ForgotPassword(ctx context.Context, in ForgotPasswordInput) (*ForgotPasswordOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" {
		return nil, userdomain.ErrEmailRequired
	}

	profile, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, userdomain.ErrNotFound
	}

	token, err := generateResetToken()
	if err != nil {
		return nil, fmt.Errorf("auth.ForgotPassword: generate token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(passwordResetTTL)
	if err := s.resetTokenRepo.Create(ctx, profile.ID(), token, expiresAt); err != nil {
		return nil, fmt.Errorf("auth.ForgotPassword: %w", err)
	}

	resetLink := fmt.Sprintf("%s/#/reset-password?token=%s", frontendBaseURL(), token)

	// ВРЕМЕННО вместо реальной отправки письма — см. заметку в доке метода выше.
	log.Printf("[auth] password reset requested for %s — link: %s (expires %s)", profile.Email(), resetLink, expiresAt.Format(time.RFC3339))

	return &ForgotPasswordOutput{
		ResetLink: resetLink,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

// ResetPassword проверяет токен (существует, не использован, не истёк) и устанавливает новый пароль.
func (s *Service) ResetPassword(ctx context.Context, in ResetPasswordInput) error {
	token := strings.TrimSpace(in.Token)
	if token == "" {
		return authdomain.ErrTokenInvalid
	}

	password := strings.TrimSpace(in.Password)
	if password == "" {
		return authdomain.ErrPasswordRequired
	}
	if len([]rune(password)) < userdomain.MinPasswordLength {
		return authdomain.ErrPasswordTooShort
	}

	rt, err := s.resetTokenRepo.GetByToken(ctx, token)
	if err != nil {
		return err // уже authdomain.ErrTokenInvalid из репозитория, если токена нет
	}
	if rt.UsedAt != nil {
		return authdomain.ErrTokenInvalid
	}
	if time.Now().UTC().After(rt.ExpiresAt) {
		return authdomain.ErrTokenExpired
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth.ResetPassword: bcrypt: %w", err)
	}

	if err := s.repo.UpdatePassword(ctx, rt.ProfileID, string(hash)); err != nil {
		return fmt.Errorf("auth.ResetPassword: %w", err)
	}
	if err := s.resetTokenRepo.MarkUsed(ctx, token); err != nil {
		return fmt.Errorf("auth.ResetPassword: mark used: %w", err)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// generateResetToken создаёт криптографически случайный токен (32 байта → 64 hex-символа).
func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// frontendBaseURL — адрес фронтенда для сборки ссылки восстановления. Берётся из env FRONTEND_URL,
// с фолбэком на продакшн-домен для дев-окружения без переменной.
func frontendBaseURL() string {
	if v := os.Getenv("FRONTEND_URL"); v != "" {
		return strings.TrimSuffix(v, "/")
	}
	return "https://hanbin-drama.com"
}
