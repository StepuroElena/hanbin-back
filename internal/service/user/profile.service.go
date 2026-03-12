package user

import (
	"context"
	"fmt"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

// Service реализует прикладные use-case'ы.
// Зависит от репозиториев: профилей, пользователей, дорам и бэйджей.
type Service struct {
	repo       domain.Repository
	userRepo   domain.UserRepository
	dramaRepo  domain.DramaRepository
	badgeRepo  domain.BadgeRepository
}

// NewService — конструктор с внедрением зависимостей.
func NewService(repo domain.Repository, userRepo domain.UserRepository, dramaRepo domain.DramaRepository, badgeRepo domain.BadgeRepository) *Service {
	return &Service{repo: repo, userRepo: userRepo, dramaRepo: dramaRepo, badgeRepo: badgeRepo}
}

// ── DTO ──────────────────────────────────────────────────────────────────────

// CreateInput — входные данные для создания профиля.
type CreateInput struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
}

// UpdateInput — данные для обновления профиля. Пустая строка = не менять.
type UpdateInput struct {
	Name string `json:"name"`
}

// ProfileOutput — то, что возвращается наружу (handler / API).
type ProfileOutput struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ── Use cases ─────────────────────────────────────────────────────────────────

// Create создаёт профиль для существующего пользователя.
// Проверяет, что пользователь с таким user_id существует.
func (s *Service) Create(ctx context.Context, in CreateInput) (*ProfileOutput, error) {
	// Убеждаемся, что пользователь существует
	if _, err := s.userRepo.GetByID(ctx, in.UserID); err != nil {
		return nil, fmt.Errorf("service.Create: %w", domain.ErrUserNotFound)
	}

	profile, err := domain.NewProfile(in.UserID, in.Name)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	id, err := s.repo.Create(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	out := toOutput(domain.Reconstitute(id, profile.UserID(), profile.Name(), profile.CreatedAt(), profile.UpdatedAt()))
	return &out, nil
}

// GetByID возвращает профиль по ID профиля.
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
		UserID:    p.UserID(),
		Name:      p.Name(),
		CreatedAt: p.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: p.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	}
}
