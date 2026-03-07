package user

import (
	"context"
	"fmt"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

// Service реализует прикладные use-case'ы для работы с профилем пользователя.
// Зависит только от интерфейса domain.Repository — конкретная БД не важна.
type Service struct {
	repo domain.Repository
}

// NewService — конструктор с внедрением зависимости (Dependency Injection).
func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// ── DTO ──────────────────────────────────────────────────────────────────────

// CreateInput — входные данные для создания профиля.
type CreateInput struct {
	Name  string
	Email string
}

// UpdateInput — данные для обновления профиля. Пустая строка = не менять.
type UpdateInput struct {
	Name  string
	Email string
}

// ProfileOutput — то, что возвращается наружу (handler / API).
// Скрывает детали домена и позволяет независимо менять представление.
type ProfileOutput struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ── Use cases ─────────────────────────────────────────────────────────────────

// Create создаёт новый профиль пользователя.
func (s *Service) Create(ctx context.Context, in CreateInput) (*ProfileOutput, error) {
	profile, err := domain.NewProfile(in.Name, in.Email)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	id, err := s.repo.Create(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	out := toOutput(domain.Reconstitute(id, profile.Name(), profile.Email(), profile.CreatedAt(), profile.UpdatedAt()))
	return &out, nil
}

// GetByID возвращает профиль по ID.
func (s *Service) GetByID(ctx context.Context, id int64) (*ProfileOutput, error) {
	profile, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service.GetByID: %w", err)
	}
	out := toOutput(profile)
	return &out, nil
}

// Update обновляет поля профиля.
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (*ProfileOutput, error) {
	profile, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}

	if in.Name != "" {
		if err := profile.SetName(in.Name); err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
	}
	if in.Email != "" {
		if err := profile.SetEmail(in.Email); err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
	}

	if err := s.repo.Update(ctx, profile); err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}

	out := toOutput(profile)
	return &out, nil
}

// Delete удаляет профиль по ID.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	return nil
}

// ── Вспомогательные функции ───────────────────────────────────────────────────

func toOutput(p *domain.Profile) ProfileOutput {
	return ProfileOutput{
		ID:        p.ID(),
		Name:      p.Name(),
		Email:     p.Email(),
		CreatedAt: p.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: p.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	}
}
