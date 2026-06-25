package drama

import (
	"context"
	"fmt"

	domain "github.com/hanbin/hanbin-back/internal/domain/drama"
)

// Service реализует use-case'ы для работы с дорамами.
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// ── DTO ───────────────────────────────────────────────────────────────────────

// CreateInput — тело запроса на добавление дорамы.
type CreateInput struct {
	Title          string   `json:"title"`
	WatchURL       string   `json:"watch_url"`
	ReleaseYear    int      `json:"release_year"`
	ReleaseTag     string   `json:"release_tag"`     // "ongoing" | "released"
	TranslationTag string   `json:"translation_tag"` // "translated" | "translating"
	Genre          string   `json:"genre"`
	Rating         *float64 `json:"rating"`          // опционально
	Country        string   `json:"country"`
}

// DramaOutput — то, что возвращается клиенту.
type DramaOutput struct {
	ID             int64    `json:"id"`
	ProfileID      int64    `json:"profile_id"`
	Title          string   `json:"title"`
	WatchURL       string   `json:"watch_url"`
	ReleaseYear    int      `json:"release_year"`
	ReleaseTag     string   `json:"release_tag"`
	TranslationTag string   `json:"translation_tag"`
	Genre          string   `json:"genre"`
	Rating         *float64 `json:"rating"`
	WatchStatus    string   `json:"watch_status"`
	Country        string   `json:"country"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// ── Use cases ─────────────────────────────────────────────────────────────────

// Create добавляет новую дораму, привязанную к profileID из токена.
func (s *Service) Create(ctx context.Context, profileID int64, in CreateInput) (*DramaOutput, error) {
	releaseTag, err := domain.ParseReleaseTag(in.ReleaseTag)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	translationTag, err := domain.ParseTranslationTag(in.TranslationTag)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	d, err := domain.NewDrama(
		profileID,
		in.Title,
		in.WatchURL,
		in.ReleaseYear,
		releaseTag,
		translationTag,
		in.Genre,
		in.Rating,
		in.Country,
	)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	id, err := s.repo.Create(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	out := toOutput(domain.Reconstitute(
		id, profileID,
		d.Title(), d.WatchURL(),
		d.ReleaseYear(),
		d.ReleaseTag(), d.TranslationTag(),
		d.Genre(), d.Rating(),
		d.WatchStatus(), d.Country(),
		d.CreatedAt(), d.UpdatedAt(),
	))
	return &out, nil
}

// GetAllByProfileID возвращает все дорамы пользователя — используется в /users/me.
func (s *Service) GetAllByProfileID(ctx context.Context, profileID int64) ([]DramaOutput, error) {
	dramas, err := s.repo.GetAllByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.GetAllByProfileID: %w", err)
	}

	out := make([]DramaOutput, 0, len(dramas))
	for _, d := range dramas {
		out = append(out, toOutput(d))
	}
	return out, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toOutput(d *domain.Drama) DramaOutput {
	return DramaOutput{
		ID:             d.ID(),
		ProfileID:      d.ProfileID(),
		Title:          d.Title(),
		WatchURL:       d.WatchURL(),
		ReleaseYear:    d.ReleaseYear(),
		ReleaseTag:     string(d.ReleaseTag()),
		TranslationTag: string(d.TranslationTag()),
		Genre:          d.Genre(),
		Rating:         d.Rating(),
		WatchStatus:    string(d.WatchStatus()),
		Country:        d.Country(),
		CreatedAt:      d.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      d.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	}
}
