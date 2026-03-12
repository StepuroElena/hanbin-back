package user

import (
	"context"
	"fmt"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

// Service реализует use-case'ы для работы с профилем пользователя.
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// ── DTO ───────────────────────────────────────────────────────────────────────

// CreateInput оставлен для совместимости с /api/v1/profiles (прямое создание без пароля).
type CreateInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UpdateInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ProfileOutput — публичное представление профиля.
type ProfileOutput struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ── Use cases ─────────────────────────────────────────────────────────────────

// Create создаёт профиль без пароля (для прямого API без авторизации).
// Используется только в /api/v1/profiles POST.
func (s *Service) Create(ctx context.Context, in CreateInput) (*ProfileOutput, error) {
	// Пустой password_hash — только для legacy эндпоинта
	profile, err := domain.NewProfile(in.Name, in.Email, "no-password")
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	id, err := s.repo.Create(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	out := toOutput(domain.Reconstitute(id, profile.Name(), profile.Email(), "", profile.CreatedAt(), profile.UpdatedAt()))
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

// ── helpers ───────────────────────────────────────────────────────────────────

func toOutput(p *domain.Profile) ProfileOutput {
	return ProfileOutput{
		ID:        p.ID(),
		Name:      p.Name(),
		Email:     p.Email(),
		CreatedAt: p.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: p.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	}
}
