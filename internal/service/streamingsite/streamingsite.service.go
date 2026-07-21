package streamingsite

import (
	"context"
	"fmt"

	domain "github.com/hanbin/hanbin-back/internal/domain/streamingsite"
)

// Service реализует use-case'ы для работы с сайтами просмотра.
type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// defaultSite — один элемент дефолтного набора, с которым стартует каждый профиль.
type defaultSite struct {
	name     string
	url      string
	language domain.Language
}

// defaultSites — тот же список, что раньше был захардкожен во фронте (STREAMING_SITES
// в AddDramaModal.js). Сажается лениво в GetAllByProfileID, если у профиля ещё нет ни одного сайта —
// так подхватываются и новые регистрации, и уже существующие профили без миграции данных.
// Все дефолтные сайты создаются включёнными (enabled=true) — см. domain.NewStreamingSite.
var defaultSites = []defaultSite{
	{name: "DoramaTV", url: "https://m.doramatv.one", language: domain.LanguageRU},
	{name: "Dorama.land", url: "https://dorama.land", language: domain.LanguageRU},
	{name: "Doramy.club", url: "https://doramy.club", language: domain.LanguageRU},
	{name: "Doramy.info", url: "https://doramy.info", language: domain.LanguageRU},
	{name: "Doramiru", url: "https://doram-ru.org", language: domain.LanguageRU},
	{name: "Dorama24", url: "https://dorama24.su", language: domain.LanguageRU},
	{name: "Rakuten Viki", url: "https://viki.com", language: domain.LanguageEN},
	{name: "Netflix", url: "https://netflix.com", language: domain.LanguageMulti},
	{name: "iQiyi", url: "https://iq.com", language: domain.LanguageMulti},
	{name: "MyDramaList", url: "https://mydramalist.com", language: domain.LanguageEN},
}

// ── DTO ───────────────────────────────────────────────────────────────────────

// SiteOutput — публичное представление сайта, отдаётся клиенту.
type SiteOutput struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Language string `json:"language"`
	Position int    `json:"position"`
	Enabled  bool   `json:"enabled"`
}

// CreateInput — тело запроса на добавление собственного сайта.
type CreateInput struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Language string `json:"language"` // опционально, дефолт "ru"
}

// UpdateInput — тело запроса на обновление сайта. Все поля опциональны.
type UpdateInput struct {
	Name     *string `json:"name"`
	URL      *string `json:"url"`
	Language *string `json:"language"`
	Position *int    `json:"position"`
	Enabled  *bool   `json:"enabled"`
}

// ── Use cases ─────────────────────────────────────────────────────────────────

// GetAllByProfileID возвращает список сайтов профиля. Если у профиля ещё нет ни одного
// сайта (новая регистрация или старый профиль без миграции) — досеивает дефолтный набор.
func (s *Service) GetAllByProfileID(ctx context.Context, profileID int64) ([]SiteOutput, error) {
	count, err := s.repo.CountByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.GetAllByProfileID: %w", err)
	}
	if count == 0 {
		if err := s.seedDefaults(ctx, profileID); err != nil {
			return nil, fmt.Errorf("service.GetAllByProfileID seed: %w", err)
		}
	}

	sites, err := s.repo.GetAllByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.GetAllByProfileID: %w", err)
	}

	out := make([]SiteOutput, 0, len(sites))
	for _, site := range sites {
		out = append(out, toOutput(site))
	}
	return out, nil
}

// Create добавляет пользователю собственный сайт (за пределами дефолтного набора). Включён по умолчанию.
func (s *Service) Create(ctx context.Context, profileID int64, in CreateInput) (*SiteOutput, error) {
	language, err := domain.ParseLanguage(in.Language)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	// Новый сайт уходит в конец списка — считаем текущее количество как позицию.
	count, err := s.repo.CountByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	site, err := domain.NewStreamingSite(profileID, in.Name, in.URL, language, count)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	id, err := s.repo.Create(ctx, site)
	if err != nil {
		return nil, fmt.Errorf("service.Create: %w", err)
	}

	out := toOutput(domain.Reconstitute(id, profileID, site.Name(), site.URL(), site.Language(), site.Position(), site.Enabled(), site.CreatedAt(), site.UpdatedAt()))
	return &out, nil
}

// Update применяет частичное обновление сайта. Проверяет принадлежность профилю из токена.
func (s *Service) Update(ctx context.Context, profileID, siteID int64, in UpdateInput) (*SiteOutput, error) {
	site, err := s.repo.GetByID(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}
	if site.ProfileID() != profileID {
		return nil, fmt.Errorf("service.Update: %w", domain.ErrNotFound)
	}

	if in.Name != nil {
		if err := site.SetName(*in.Name); err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
	}
	if in.URL != nil {
		if err := site.SetURL(*in.URL); err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
	}
	if in.Language != nil {
		language, err := domain.ParseLanguage(*in.Language)
		if err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
		if err := site.SetLanguage(language); err != nil {
			return nil, fmt.Errorf("service.Update: %w", err)
		}
	}
	if in.Position != nil {
		site.SetPosition(*in.Position)
	}
	if in.Enabled != nil {
		site.SetEnabled(*in.Enabled)
	}

	if err := s.repo.Update(ctx, site); err != nil {
		return nil, fmt.Errorf("service.Update: %w", err)
	}

	out := toOutput(site)
	return &out, nil
}

// Delete удаляет сайт пользователя. Проверяет принадлежность профилю из токена.
func (s *Service) Delete(ctx context.Context, profileID, siteID int64) error {
	site, err := s.repo.GetByID(ctx, siteID)
	if err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	if site.ProfileID() != profileID {
		return fmt.Errorf("service.Delete: %w", domain.ErrNotFound)
	}
	if err := s.repo.Delete(ctx, siteID); err != nil {
		return fmt.Errorf("service.Delete: %w", err)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (s *Service) seedDefaults(ctx context.Context, profileID int64) error {
	for i, d := range defaultSites {
		site, err := domain.NewStreamingSite(profileID, d.name, d.url, d.language, i)
		if err != nil {
			return err
		}
		if _, err := s.repo.Create(ctx, site); err != nil {
			return err
		}
	}
	return nil
}

func toOutput(s *domain.StreamingSite) SiteOutput {
	return SiteOutput{
		ID:       s.ID(),
		Name:     s.Name(),
		URL:      s.URL(),
		Language: string(s.Language()),
		Position: s.Position(),
		Enabled:  s.Enabled(),
	}
}
