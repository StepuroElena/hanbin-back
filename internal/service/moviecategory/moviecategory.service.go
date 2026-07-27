package moviecategory

import (
	"context"
	"fmt"

	domain "github.com/hanbin/hanbin-back/internal/domain/moviecategory"
)

// Service реализует use-case'ы для работы с категориями фильмов.
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// defaultCategories — стартовый набор, с которым сажается каждый новый профиль —
// короткие слова/теги, описывающие фильм. Сеется лениво в GetAllByProfileID, если у профиля
// ещё нет ни одной категории — так же, как и defaultSites у streaming_sites.
var defaultCategories = []string{
	"Для вечера",
	"С попкорном",
	"Атмосферное",
	"Экранизация",
	"Культовое",
	"Заставляет задуматься",
	"Хочется поплакать",
	"Пересматриваю",
}

// ── DTO ───────────────────────────────────────────────────────────────────────

// CategoryOutput — публичное представление категории, отдаётся клиенту.
type CategoryOutput struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
	Enabled  bool   `json:"enabled"`
}

// CreateInput — тело запроса на добавление своей категории.
type CreateInput struct {
	Name string `json:"name"`
}

// UpdateInput — тело запроса на обновление категории. Все поля опциональны.
type UpdateInput struct {
	Name     *string `json:"name"`
	Position *int    `json:"position"`
	Enabled  *bool   `json:"enabled"`
}

// ── Use cases ─────────────────────────────────────────────────────────────────

// GetAllByProfileID возвращает список категорий профиля. Если у профиля ещё нет ни одной
// категории (новая регистрация или старый профиль без миграции) — досеивает дефолтный набор.
func (s *Service) GetAllByProfileID(ctx context.Context, profileID int64) ([]CategoryOutput, error) {
	count, err := s.repo.CountByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.GetAllByProfileID: %w", err)
	}
	if count == 0 {
		if err := s.seedDefaults(ctx, profileID); err != nil {
			return nil, fmt.Errorf("service.GetAllByProfileID seed: %w", err)
		}
	}

	categories, err := s.repo.GetAllByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.GetAllByProfileID: %w", err)
	}

	out := make([]CategoryOutput, 0, len(categories))
	for _, c := range categories {
		out = append(out, toOutput(c))
	}
	return out, nil
}

// Create добавляет пользователю собственную категорию (за пределами дефолтного набора). Включена по умолчанию.
func (s *Service) Create(ctx context.Context, profileID int64, in CreateInput) (*CategoryOutput, error) {
	// Новая категория уходит в конец списка — считаем текущее количество как позицию.
	count, err := s.repo.CountByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	category, err := domain.NewMovieCategory(profileID, in.Name, count)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	id, err := s.repo.Create(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	out := toOutput(domain.Reconstitute(id, profileID, category.Name(), category.Position(), category.Enabled(), category.CreatedAt(), category.UpdatedAt()))
	return &out, nil
}

// Update применяет частичное обновление категории. Проверяет принадлежность профилю из токена.
func (s *Service) Update(ctx context.Context, profileID, categoryID int64, in UpdateInput) (*CategoryOutput, error) {
	category, err := s.repo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}
	if category.ProfileID() != profileID {
		return nil, fmt.Errorf("service.Update: %w", domain.ErrNotFound)
	}

	if in.Name != nil {
		if err := category.SetName(*in.Name); err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
	}
	if in.Position != nil {
		category.SetPosition(*in.Position)
	}
	if in.Enabled != nil {
		category.SetEnabled(*in.Enabled)
	}

	if err := s.repo.Update(ctx, category); err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}

	out := toOutput(category)
	return &out, nil
}

// Delete удаляет категорию пользователя. Проверяет принадлежность профилю из токена.
func (s *Service) Delete(ctx context.Context, profileID, categoryID int64) error {
	category, err := s.repo.GetByID(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	if category.ProfileID() != profileID {
		return fmt.Errorf("service.Delete: %w", domain.ErrNotFound)
	}
	if err := s.repo.Delete(ctx, categoryID); err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (s *Service) seedDefaults(ctx context.Context, profileID int64) error {
	for i, name := range defaultCategories {
		category, err := domain.NewMovieCategory(profileID, name, i)
		if err != nil {
			return err
		}
		if _, err := s.repo.Create(ctx, category); err != nil {
			return err
		}
	}
	return nil
}

func toOutput(c *domain.MovieCategory) CategoryOutput {
	return CategoryOutput{
		ID:       c.ID(),
		Name:     c.Name(),
		Position: c.Position(),
		Enabled:  c.Enabled(),
	}
}
