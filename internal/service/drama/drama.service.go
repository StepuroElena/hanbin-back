package drama

import (
	"context"
	"fmt"
	"strings"

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

// SeasonOutput — один сезон дорамы в ответе API.
type SeasonOutput struct {
	SeasonNumber int `json:"season_number"`
	EpisodeCount int `json:"episode_count"`
}

// SeasonProgressOutput — прогресс просмотра по одному сезону в ответе API.
type SeasonProgressOutput struct {
	SeasonNumber    int `json:"season_number"`
	WatchedEpisodes int `json:"watched_episodes"`
}

// ProgressOutput — полный прогресс просмотра в ответе API.
type ProgressOutput struct {
	CurrentEpisode int                    `json:"current_episode"`
	Seasons        []SeasonProgressOutput `json:"seasons"`
}

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

// ArchiveInput — тело запроса на изменение статуса архива.
type ArchiveInput struct {
	IsArchived bool `json:"is_archived"`
}

// SeasonInput — один сезон в запросе на обновление.
type SeasonInput struct {
	SeasonNumber int `json:"season_number"`
	EpisodeCount int `json:"episode_count"`
}

// SeasonProgressInput — прогресс по сезону в запросе на обновление.
type SeasonProgressInput struct {
	SeasonNumber    int `json:"season_number"`
	WatchedEpisodes int `json:"watched_episodes"`
}

// ProgressInput — прогресс просмотра в запросе на обновление.
type ProgressInput struct {
	CurrentEpisode int                  `json:"current_episode"`
	Seasons        []SeasonProgressInput `json:"seasons"`
}

// UpdateInput — тело запроса на обновление дорамы. Все поля опциональны —
// передаётся только то, что нужно изменить.
type UpdateInput struct {
	Title              *string        `json:"title"`
	WatchURL           *string        `json:"watch_url"`
	ReleaseYear        *int           `json:"release_year"`
	ReleaseTag         *string        `json:"release_tag"`
	TranslationTag     *string        `json:"translation_tag"`
	Genre              *string        `json:"genre"`
	Rating             *float64       `json:"rating"`      // null = сбросить
	ClearRating        bool           `json:"clear_rating"` // true = убрать рейтинг
	WatchStatus        *string        `json:"watch_status"`
	Country            *string        `json:"country"`
	EpisodeDurationMin *int           `json:"episode_duration_min"` // null = сбросить
	Seasons            []SeasonInput  `json:"seasons"`
	Progress           *ProgressInput `json:"progress"`
}

// DramaOutput — то, что возвращается клиенту.
type DramaOutput struct {
	ID                 int64          `json:"id"`
	ProfileID          int64          `json:"profile_id"`
	Title              string         `json:"title"`
	WatchURL           string         `json:"watch_url"`
	ReleaseYear        int            `json:"release_year"`
	ReleaseTag         string         `json:"release_tag"`
	TranslationTag     string         `json:"translation_tag"`
	Genre              string         `json:"genre"`
	Rating             *float64       `json:"rating"`
	WatchStatus        string         `json:"watch_status"`
	Country            string         `json:"country"`
	IsArchived         bool           `json:"is_archived"`
	EpisodeDurationMin *int           `json:"episode_duration_min"`
	Seasons            []SeasonOutput `json:"seasons"`
	Progress           ProgressOutput `json:"progress"`
	CreatedAt          string         `json:"created_at"`
	UpdatedAt          string         `json:"updated_at"`
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
		d.IsArchived(), d.EpisodeDurationMin(),
		d.Seasons(), d.Progress(),
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

// SetArchived устанавливает флаг is_archived у дорамы.
// Проверяет что дорама принадлежит profileID из токена.
func (s *Service) SetArchived(ctx context.Context, profileID, dramaID int64, isArchived bool) (*DramaOutput, error) {
	d, err := s.repo.GetByID(ctx, dramaID)
	if err != nil {
		return nil, fmt.Errorf("service.SetArchived: %w", err)
	}
	if d.ProfileID() != profileID {
		return nil, fmt.Errorf("service.SetArchived: %w", domain.ErrNotFound)
	}

	if err := s.repo.UpdateArchived(ctx, dramaID, isArchived); err != nil {
		return nil, fmt.Errorf("service.SetArchived: %w", err)
	}

	// Перечитываем актуальное состояние из БД
	updated, err := s.repo.GetByID(ctx, dramaID)
	if err != nil {
		return nil, fmt.Errorf("service.SetArchived refetch: %w", err)
	}
	out := toOutput(updated)
	return &out, nil
}

// Update применяет парциальное обновление дорамы. Меняются только переданные поля.
// Проверяет принадлежность дорамы profileID из токена.
func (s *Service) Update(ctx context.Context, profileID, dramaID int64, in UpdateInput) (*DramaOutput, error) {
	d, err := s.repo.GetByID(ctx, dramaID)
	if err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}
	if d.ProfileID() != profileID {
		return nil, fmt.Errorf("service.Update: %w", domain.ErrNotFound)
	}

	// Применяем изменения поверх текущего состояния
	title          := d.Title()
	watchURL       := d.WatchURL()
	releaseYear    := d.ReleaseYear()
	releaseTag     := d.ReleaseTag()
	translTag      := d.TranslationTag()
	genre          := d.Genre()
	rating         := d.Rating()
	watchStatus    := d.WatchStatus()
	country        := d.Country()
	duration       := d.EpisodeDurationMin()
	seasons        := d.Seasons()
	progress       := d.Progress()

	if in.Title != nil {
		title = *in.Title
	}
	if in.WatchURL != nil {
		watchURL = *in.WatchURL
	}
	if in.ReleaseYear != nil {
		releaseYear = *in.ReleaseYear
	}
	if in.ReleaseTag != nil {
		parsed, err := domain.ParseReleaseTag(*in.ReleaseTag)
		if err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
		releaseTag = parsed
	}
	if in.TranslationTag != nil {
		parsed, err := domain.ParseTranslationTag(*in.TranslationTag)
		if err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
		translTag = parsed
	}
	if in.Genre != nil {
		genre = *in.Genre
	}
	if in.ClearRating {
		rating = nil
	} else if in.Rating != nil {
		rating = in.Rating
	}
	if in.WatchStatus != nil {
		parsed, err := domain.ParseWatchStatus(*in.WatchStatus)
		if err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
		watchStatus = parsed
	}
	if in.Country != nil {
		country = *in.Country
	}
	if in.EpisodeDurationMin != nil {
		if *in.EpisodeDurationMin <= 0 {
			return nil, fmt.Errorf("service.Update: %w", domain.ErrInvalidEpisodeDuration)
		}
		duration = in.EpisodeDurationMin
	}
	if in.Seasons != nil {
		mapped := make([]domain.Season, 0, len(in.Seasons))
		for _, s := range in.Seasons {
			mapped = append(mapped, domain.Season{
				SeasonNumber: s.SeasonNumber,
				EpisodeCount: s.EpisodeCount,
			})
		}
		seasons = mapped
	}
	if in.Progress != nil {
		progSeasons := make([]domain.SeasonProgress, 0, len(in.Progress.Seasons))
		for _, sp := range in.Progress.Seasons {
			progSeasons = append(progSeasons, domain.SeasonProgress{
				SeasonNumber:    sp.SeasonNumber,
				WatchedEpisodes: sp.WatchedEpisodes,
			})
		}
		progress = domain.Progress{
			CurrentEpisode: in.Progress.CurrentEpisode,
			Seasons:        progSeasons,
		}
	}

	// Собираем обновлённый агрегат через Reconstitute (валидация вынесена на уровень сервиса)
	updated := domain.Reconstitute(
		d.ID(), d.ProfileID(),
		title, watchURL,
		releaseYear,
		releaseTag, translTag,
		genre, rating,
		watchStatus, country,
		d.IsArchived(), duration,
		seasons, progress,
		d.CreatedAt(), d.UpdatedAt(),
	)

	// Валидация обязательных полей
	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("service.Update: %w", domain.ErrTitleRequired)
	}
	if strings.TrimSpace(watchURL) == "" {
		return nil, fmt.Errorf("service.Update: %w", domain.ErrWatchURLRequired)
	}
	if strings.TrimSpace(genre) == "" {
		return nil, fmt.Errorf("service.Update: %w", domain.ErrGenreRequired)
	}
	if strings.TrimSpace(country) == "" {
		return nil, fmt.Errorf("service.Update: %w", domain.ErrCountryRequired)
	}
	if releaseYear < domain.MinYear || releaseYear > domain.MaxYear {
		return nil, fmt.Errorf("service.Update: %w", domain.ErrInvalidYear)
	}
	if rating != nil && (*rating < domain.MinRating || *rating > domain.MaxRating) {
		return nil, fmt.Errorf("service.Update: %w", domain.ErrInvalidRating)
	}

	if err := s.repo.Update(ctx, updated); err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}

	// Перечитываем актуальное состояние из БД
	fresh, err := s.repo.GetByID(ctx, dramaID)
	if err != nil {
		return nil, fmt.Errorf("service.Update refetch: %w", err)
	}
	out := toOutput(fresh)
	return &out, nil
}

// Delete проверяет, что дорама архивирована, и удаляет её из БД.
// Если is_archived = false — возвращает domain.ErrNotArchived (400).
func (s *Service) Delete(ctx context.Context, profileID, dramaID int64) error {
	d, err := s.repo.GetByID(ctx, dramaID)
	if err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	if d.ProfileID() != profileID {
		return fmt.Errorf("service.Delete: %w", domain.ErrNotFound)
	}
	if !d.IsArchived() {
		return fmt.Errorf("service.Delete: %w", domain.ErrNotArchived)
	}
	if err := s.repo.Delete(ctx, dramaID); err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toOutput(d *domain.Drama) DramaOutput {
	seasons := make([]SeasonOutput, 0, len(d.Seasons()))
	for _, s := range d.Seasons() {
		seasons = append(seasons, SeasonOutput{
			SeasonNumber: s.SeasonNumber,
			EpisodeCount: s.EpisodeCount,
		})
	}

	prog := d.Progress()
	progressSeasons := make([]SeasonProgressOutput, 0, len(prog.Seasons))
	for _, sp := range prog.Seasons {
		progressSeasons = append(progressSeasons, SeasonProgressOutput{
			SeasonNumber:    sp.SeasonNumber,
			WatchedEpisodes: sp.WatchedEpisodes,
		})
	}

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
		IsArchived:     d.IsArchived(),
		EpisodeDurationMin: d.EpisodeDurationMin(),
		Seasons:        seasons,
		Progress: ProgressOutput{
			CurrentEpisode: prog.CurrentEpisode,
			Seasons:        progressSeasons,
		},
		CreatedAt: d.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: d.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	}
}
