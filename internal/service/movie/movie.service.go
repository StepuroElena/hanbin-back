package movie

import (
	"context"
	"fmt"

	domain "github.com/hanbin/hanbin-back/internal/domain/movie"
)

// Service реализует use-case'ы для работы с фильмами.
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// ── DTO ───────────────────────────────────────────────────────────────────────

// CreateInput — тело запроса на добавление фильма.
type CreateInput struct {
	Title       string `json:"title"`
	Genre       string `json:"genre"`
	Country     string `json:"country"` // опционально
	Category    string `json:"category"` // опционально — значение из персонального списка movie_categories
	ReleaseYear *int   `json:"release_year"` // опционально
}

// UpdateStatusInput — тело запроса на изменение статуса просмотра.
type UpdateStatusInput struct {
	WatchStatus string `json:"watch_status"` // "planned" | "watched"
}

// MovieOutput — то, что возвращается клиенту.
type MovieOutput struct {
	ID          int64  `json:"id"`
	ProfileID   int64  `json:"profile_id"`
	Title       string `json:"title"`
	Genre       string `json:"genre"`
	Country     string `json:"country"`
	Category    string `json:"category"`
	ReleaseYear *int   `json:"release_year"`
	WatchStatus string `json:"watch_status"`
	IsArchived  bool   `json:"is_archived"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// StatsOutput — счётчики для карточек на странице фильмов.
type StatsOutput struct {
	MoviesWatched int `json:"movies_watched"`
	MoviesPlanned int `json:"movies_planned"`
}

// ── Use cases ─────────────────────────────────────────────────────────────────

// Create добавляет новый фильм, привязанный к profileID из токена.
func (s *Service) Create(ctx context.Context, profileID int64, in CreateInput) (*MovieOutput, error) {
	m, err := domain.NewMovie(profileID, in.Title, in.Genre, in.Country, in.Category, in.ReleaseYear)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	id, err := s.repo.Create(ctx, m)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	out := toOutput(domain.Reconstitute(id, profileID, m.Title(), m.Genre(), m.Country(), m.Category(), m.ReleaseYear(), m.WatchStatus(), m.IsArchived(), m.CreatedAt(), m.UpdatedAt()))
	return &out, nil
}

// GetAllByProfileID возвращает ВСЕ фильмы пользователя, включая архивированные — фронт сам
// фильтрует по is_archived (так же, как и с дорамами) — используется в GET /api/v1/movies.
func (s *Service) GetAllByProfileID(ctx context.Context, profileID int64) ([]MovieOutput, error) {
	movies, err := s.repo.GetAllByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.GetAllByProfileID: %w", err)
	}

	out := make([]MovieOutput, 0, len(movies))
	for _, m := range movies {
		out = append(out, toOutput(m))
	}
	return out, nil
}

// GetStats возвращает счётчики просмотренных/запланированных фильмов пользователя.
// Архивированные фильмы не учитываются — так же, как и у дорам.
func (s *Service) GetStats(ctx context.Context, profileID int64) (*StatsOutput, error) {
	movies, err := s.repo.GetAllByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.GetStats: %w", err)
	}

	stats := &StatsOutput{}
	for _, m := range movies {
		if m.IsArchived() {
			continue
		}
		switch m.WatchStatus() {
		case domain.WatchStatusCompleted:
			stats.MoviesWatched++
		case domain.WatchStatusPlanned:
			stats.MoviesPlanned++
		}
	}
	return stats, nil
}

// UpdateStatus меняет статус просмотра фильма. Проверяет, что фильм принадлежит
// profileID из токена — так же, как SetArchived у дорам.
func (s *Service) UpdateStatus(ctx context.Context, profileID, movieID int64, in UpdateStatusInput) (*MovieOutput, error) {
	m, err := s.repo.GetByID(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("service.UpdateStatus: %w", err)
	}
	if m.ProfileID() != profileID {
		return nil, fmt.Errorf("service.UpdateStatus: %w", domain.ErrNotFound)
	}

	status, err := domain.ParseWatchStatus(in.WatchStatus)
	if err != nil {
		return nil, fmt.Errorf("service.UpdateStatus: %w", err)
	}

	if err := s.repo.UpdateWatchStatus(ctx, movieID, status); err != nil {
		return nil, fmt.Errorf("service.UpdateStatus: %w", err)
	}

	updated, err := s.repo.GetByID(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("service.UpdateStatus refetch: %w", err)
	}
	out := toOutput(updated)
	return &out, nil
}

// SetArchived устанавливает флаг is_archived у фильма. Проверяет, что фильм принадлежит
// profileID из токена — так же, как SetArchived у дорам.
func (s *Service) SetArchived(ctx context.Context, profileID, movieID int64, isArchived bool) (*MovieOutput, error) {
	m, err := s.repo.GetByID(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("service.SetArchived: %w", err)
	}
	if m.ProfileID() != profileID {
		return nil, fmt.Errorf("service.SetArchived: %w", domain.ErrNotFound)
	}

	if err := s.repo.UpdateArchived(ctx, movieID, isArchived); err != nil {
		return nil, fmt.Errorf("service.SetArchived: %w", err)
	}

	updated, err := s.repo.GetByID(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("service.SetArchived refetch: %w", err)
	}
	out := toOutput(updated)
	return &out, nil
}

// Delete проверяет, что фильм архивирован, и удаляет его из БД. Если is_archived = false —
// возвращает domain.ErrNotArchived (400) — так же, как и у дорам.
func (s *Service) Delete(ctx context.Context, profileID, movieID int64) error {
	m, err := s.repo.GetByID(ctx, movieID)
	if err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	if m.ProfileID() != profileID {
		return fmt.Errorf("service.Delete: %w", domain.ErrNotFound)
	}
	if !m.IsArchived() {
		return fmt.Errorf("service.Delete: %w", domain.ErrNotArchived)
	}
	if err := s.repo.Delete(ctx, movieID); err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toOutput(m *domain.Movie) MovieOutput {
	return MovieOutput{
		ID:          m.ID(),
		ProfileID:   m.ProfileID(),
		Title:       m.Title(),
		Genre:       m.Genre(),
		Country:     m.Country(),
		Category:    m.Category(),
		ReleaseYear: m.ReleaseYear(),
		WatchStatus: string(m.WatchStatus()),
		IsArchived:  m.IsArchived(),
		CreatedAt:   m.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   m.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	}
}
