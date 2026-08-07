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
	repo                  userdomain.Repository
	resetTokenRepo        authdomain.ResetTokenRepository
	confirmationTokenRepo authdomain.ConfirmationTokenRepository
	mailer                Mailer
	allowedOrigins        []string
}

// Mailer — минимальный интерфейс отправки писем, чтобы сервис не зависел напрямую от
// конкретного провайдера (см. internal/mailer.ResendMailer, он реализует этот интерфейс
// структурно, без явной зависимости от этого пакета).
type Mailer interface {
	Enabled() bool
	Send(to, subject, htmlBody string) error
}

// allowedOrigins — тот же список, что и у CORS-middleware (ALLOWED_ORIGINS из .env) — используется
// в ForgotPassword, чтобы ссылка восстановления вела на тот хост, с которого реально пришёл запрос.
func NewService(repo userdomain.Repository, resetTokenRepo authdomain.ResetTokenRepository, confirmationTokenRepo authdomain.ConfirmationTokenRepository, mailer Mailer, allowedOrigins []string) *Service {
	return &Service{repo: repo, resetTokenRepo: resetTokenRepo, confirmationTokenRepo: confirmationTokenRepo, mailer: mailer, allowedOrigins: allowedOrigins}
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

	// ConfirmationLink заполнен ТОЛЬКО когда мейлер не настроен или отправка провалилась — фронт
	// в этом случае показывает ссылку прямо в UI (см. RegisterModal.js), тот же паттерн, что и у
	// ForgotPasswordOutput.ResetLink. Когда письмо реально ушло — поле пустое.
	ConfirmationLink string `json:"confirmation_link,omitempty"`
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
// ResetLink заполнен только когда мейлер не настроен или отправка провалилась — фронт в этом случае показывает
// ссылку прямо в UI (см. ForgotPasswordModal.js). Когда письмо реально ушло — поле пустое.
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
const emailConfirmationTTL = 24 * time.Hour

// ── Use cases ─────────────────────────────────────────────────────────────────

// Register создаёт нового пользователя с хешированным паролем и отправляет письмо с подтверждением email.
// Аккаунт создаётся сразу (вход ничем не блокируется до подтверждения почты — сознательное решение,
// чтобы не усложнять первый вход), но Profile.IsEmailConfirmed() остаётся false, пока пользователь
// не перейдёт по ссылке из письма (см. ConfirmEmail).
//
// requestOrigin — тот же заголовок Origin из HTTP-запроса, что и в ForgotPassword — чтобы ссылка
// подтверждения вела на тот хост, с которого реально пришёл запрос.
func (s *Service) Register(ctx context.Context, in RegisterInput, requestOrigin string) (*RegisterOutput, error) {
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

	out := &RegisterOutput{
		UserID: id,
		Name:   profile.Name(),
		Email:  profile.Email(),
	}

	// Отправка письма подтверждения не должна ронять регистрацию — аккаунт уже создан и валиден,
	// сбой почты логируется, пользователь всё равно получает успешный ответ.
	//
	// Важно: передаём id явно, а не через profile.ID() — profile это тот же объект,
	// что вернул конструктор NewProfile() и его id всё ещё 0 (Create возвращает
	// присвоенный БД id отдельно, не мутируя сам profile). С profile.ID() токен бы сохранялся
	// с profile_id = 0 и ломал бы FK-ограничение (что и происходило до этого фикса).
	if err := s.sendConfirmationEmail(ctx, id, profile, requestOrigin, out); err != nil {
		log.Printf("[auth] failed to send confirmation email to %s: %v", profile.Email(), err)
	}

	return out, nil
}

// sendConfirmationEmail генерирует токен подтверждения (живёт emailConfirmationTTL) и шлёт письмо со ссылкой.
// Если мейлер не настроен или отправка сорвалась — тот же фолбэк, что и в ForgotPassword: ссылка
// кладётся в out.ConfirmationLink, чтобы локальная разработка без ключа не ломалась.
func (s *Service) sendConfirmationEmail(ctx context.Context, profileID int64, profile *userdomain.Profile, requestOrigin string, out *RegisterOutput) error {
	token, err := generateResetToken() // тот же генератор случайных токенов, что и у ForgotPassword
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(emailConfirmationTTL)
	if err := s.confirmationTokenRepo.Create(ctx, profileID, token, expiresAt); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	base := frontendBaseURL()
	if requestOrigin != "" && middleware.IsAllowedOrigin(requestOrigin, s.allowedOrigins) {
		base = strings.TrimSuffix(requestOrigin, "/")
	}
	confirmLink := fmt.Sprintf("%s/#/confirm-email?token=%s", base, token)

	if s.mailer != nil && s.mailer.Enabled() {
		subject := "Подтверди почту — Hanbin"
		html := fmt.Sprintf(`<p>Спасибо за регистрацию в Hanbin! Осталось подтвердить почту.</p>
<p><a href="%s">Подтвердить email</a></p>
<p>Ссылка действительна 24 часа. Если это были не вы — просто проигнорируйте это письмо.</p>`, confirmLink)

		if err := s.mailer.Send(profile.Email(), subject, html); err != nil {
			// Не роняем весь запрос из-за сбоя почты — токен уже создан и валиден, отдаём ссылку в ответе как fallback.
			log.Printf("[auth] resend error for %s: %v (link: %s)", profile.Email(), err, confirmLink)
			out.ConfirmationLink = confirmLink
			return nil
		}
		log.Printf("[auth] confirmation email sent to %s", profile.Email())
		return nil
	}

	// Фолбэк для дев-окружения без RESEND_API_KEY — ссылка логируется и возвращается в ответе.
	log.Printf("[auth] RESEND_API_KEY not set — confirmation link for %s: %s (expires %s)", profile.Email(), confirmLink, expiresAt.Format(time.RFC3339))
	out.ConfirmationLink = confirmLink
	return nil
}

// ConfirmEmail проверяет токен подтверждения (существует, не использован, не истёк), помечает его
// использованным и ставит email_confirmed_at у профиля. Вызывается по ссылке из письма
// (#/confirm-email?token=...), см. router.js на фронте.
func (s *Service) ConfirmEmail(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return authdomain.ErrConfirmationTokenInvalid
	}

	ct, err := s.confirmationTokenRepo.GetByToken(ctx, token)
	if err != nil {
		return err // уже authdomain.ErrConfirmationTokenInvalid из репозитория, если токена нет
	}
	if ct.UsedAt != nil {
		return authdomain.ErrConfirmationTokenInvalid
	}
	if time.Now().UTC().After(ct.ExpiresAt) {
		return authdomain.ErrConfirmationTokenExpired
	}

	if err := s.repo.ConfirmEmail(ctx, ct.ProfileID); err != nil {
		return fmt.Errorf("auth.ConfirmEmail: %w", err)
	}
	if err := s.confirmationTokenRepo.MarkUsed(ctx, token); err != nil {
		return fmt.Errorf("auth.ConfirmEmail: mark used: %w", err)
	}
	return nil
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
// (живёт passwordResetTTL) и отправляет письмо со ссылкой на страницу смены пароля.
//
// Если email не найден — возвращает userdomain.ErrNotFound (404 на уровне хендлера), как явно
// просил продукт: пользователь должен видеть ошибку, если такой почты нет. Это раскрывает факт
// существования аккаунта по email (user enumeration) — сознательный компромисс по запросу продукта,
// не техническое упущение.
//
// Если мейлер настроен (RESEND_API_KEY задан) — письмо реально отправляется, и ResetLink в ответе
// остаётся пустым (ссылка больше не раскрывается через API). Если мейлера нет или отправка
// сорвалась — фолбэк на старое поведение: ссылка логируется и возвращается в ответе, чтобы локальная
// разработка без ключа не ломалась.
//
// requestOrigin — заголовок Origin из HTTP-запроса (см. handler). Если он есть и входит в список
// разрешённых (allowedOrigins, те же ALLOWED_ORIGINS, что и у CORS) — ссылка строится именно на него:
// таким образом она всегда ведёт на тот хост, с которого реально пришёл запрос (localhost в dev, прод-домен
// в проде) — без ручной настройки FRONTEND_URL для каждого окружения. Если origin пуст или не
// в списке — фолбэк на frontendBaseURL() (env FRONTEND_URL или прод-домен по умолчанию).
// Не доверяем origin вслепую — иначе кто-то мог бы подсунуть в письмо жертвы ссылку на фишинг-клон.
func (s *Service) ForgotPassword(ctx context.Context, in ForgotPasswordInput, requestOrigin string) (*ForgotPasswordOutput, error) {
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

	base := frontendBaseURL()
	if requestOrigin != "" && middleware.IsAllowedOrigin(requestOrigin, s.allowedOrigins) {
		base = strings.TrimSuffix(requestOrigin, "/")
	}
	resetLink := fmt.Sprintf("%s/#/reset-password?token=%s", base, token)

	if s.mailer != nil && s.mailer.Enabled() {
		subject := "Восстановление пароля — Hanbin"
		html := fmt.Sprintf(`<p>Кто-то (надеемся, что это вы) запросил восстановление пароля для аккаунта Hanbin.</p>
<p><a href="%s">Установить новый пароль</a></p>
<p>Ссылка действительна 1 час. Если это были не вы — просто проигнорируйте это письмо.</p>`, resetLink)

		if err := s.mailer.Send(profile.Email(), subject, html); err != nil {
			// Не роняем весь запрос из-за сбоя почты — токен уже создан и валиден, просто логируем и отдаём
			// ссылку в ответе как fallback, чтобы пользователь не остался ни с чем.
			log.Printf("[auth] failed to send reset email to %s: %v (link: %s)", profile.Email(), err, resetLink)
			return &ForgotPasswordOutput{ResetLink: resetLink, ExpiresAt: expiresAt.Format(time.RFC3339)}, nil
		}

		log.Printf("[auth] password reset email sent to %s", profile.Email())
		return &ForgotPasswordOutput{ExpiresAt: expiresAt.Format(time.RFC3339)}, nil
	}

	// Фолбэк для дев-окружения без RESEND_API_KEY — ссылка логируется и возвращается в ответе.
	log.Printf("[auth] RESEND_API_KEY not set — password reset link for %s: %s (expires %s)", profile.Email(), resetLink, expiresAt.Format(time.RFC3339))
	return &ForgotPasswordOutput{ResetLink: resetLink, ExpiresAt: expiresAt.Format(time.RFC3339)}, nil
}

// ValidateResetToken проверяет токен (существует, не использован, не истёк) и возвращает email аккаунта, чтобы
// фронт мог показать в модалке «Меняем пароль для ...» до того, как пользователь введёт новый
// пароль. Чисто чтение — ничего не меняет и не помечает токен использованным.
func (s *Service) ValidateResetToken(ctx context.Context, token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", authdomain.ErrTokenInvalid
	}

	rt, err := s.resetTokenRepo.GetByToken(ctx, token)
	if err != nil {
		return "", err // authdomain.ErrTokenInvalid из репозитория, если токена нет
	}
	if rt.UsedAt != nil {
		return "", authdomain.ErrTokenInvalid
	}
	if time.Now().UTC().After(rt.ExpiresAt) {
		return "", authdomain.ErrTokenExpired
	}

	profile, err := s.repo.GetByID(ctx, rt.ProfileID)
	if err != nil {
		return "", fmt.Errorf("auth.ValidateResetToken: %w", err)
	}
	return profile.Email(), nil
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
