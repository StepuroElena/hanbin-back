// Package random реализует фичу «Рандом» (Random Picker) на бэке: подбор случайного
// тайтла из библиотеки пользователя в статусе «Запланировано», с фильтрами по типу
// (фильм/дорама/неважно) и жанру, и защитой от повтора того же тайтла при "Ещё раз".
//
// Раньше вся эта логика (фильтрация, взвешенный выбор между дорамами и фильмами,
// подсчёт доступных жанров) жила во фронтенде (src/utils/pickRandom.js) и требовала
// выгружать в браузер всю библиотеку пользователя. Теперь фронт — тонкий клиент:
// делает один запрос и просто рендерит то, что вернул бэк.
package random

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"

	dramadomain "github.com/hanbin/hanbin-back/internal/domain/drama"
	moviedomain "github.com/hanbin/hanbin-back/internal/domain/movie"
)

// ErrNoMatch — под переданные фильтры (тип/жанр) в статусе "planned" ничего не нашлось.
var ErrNoMatch = errors.New("no matching planned title")

// Type — фильтр по типу тайтла. TypeAny означает случайный выбор между дорамами и фильмами,
// взвешенный по размеру каждого пула (а не 50/50 — иначе при 1 фильме и 50 дорамах
// фильм выпадал бы неоправданно часто).
type Type string

const (
	TypeAny    Type = "any"
	TypeMovie  Type = "movie"
	TypeSeries Type = "series"
)

func ParseType(s string) Type {
	switch Type(s) {
	case TypeMovie, TypeSeries:
		return Type(s)
	default:
		return TypeAny
	}
}

// Kind — тип конкретного результата (в отличие от Type, который может быть "любой").
type Kind string

const (
	KindDrama Kind = "drama"
	KindMovie Kind = "movie"
)

type Service struct {
	dramaRepo dramadomain.Repository
	movieRepo moviedomain.Repository
}

func NewService(dramaRepo dramadomain.Repository, movieRepo moviedomain.Repository) *Service {
	return &Service{dramaRepo: dramaRepo, movieRepo: movieRepo}
}

// FacetsOutput — ответ GET /api/v1/random/facets.
type FacetsOutput struct {
	Genres    []string `json:"genres"`     // объединённые жанры дорам+фильмов в "planned", без дублей
	HasMovies bool     `json:"has_movies"` // есть ли хоть один фильм в "planned" — фронт дизейблит кнопку "Фильм", если false
	HasSeries bool     `json:"has_series"` // то же для дорам
}

// GetFacets возвращает доступные жанры и флаги наличия каждого типа — используется
// для построения чипов в модалке выбора (шаг "picker") под текущий typeFilter.
func (s *Service) GetFacets(ctx context.Context, profileID int64, typeFilter Type) (*FacetsOutput, error) {
	var dramaGenres, movieGenres []string
	var err error

	// has_movies/has_series отражают глобальное наличие — считаем их ВСЕГДА, вне зависимости от typeFilter.
	// Если считать их только когда typeFilter разрешает этот тип — получается self-lock: если
	// фронт уже залочил тип на "series" (потому что фильмов раньше не было), следующий запрос
	// с type=series никогда больше не пересчитает фильмы — has_movies навсегда останется false, даже
	// если фильмы появятся. Поэтому счёты и жанровый список — два разных понятия: счёт всегда
	// глобальный, жанры сужены под текущий тип.
	dramaCount, err := s.dramaRepo.CountPlanned(ctx, profileID, "")
	if err != nil {
		return nil, fmt.Errorf("random service.GetFacets drama count: %w", err)
	}
	movieCount, err := s.movieRepo.CountPlanned(ctx, profileID, "")
	if err != nil {
		return nil, fmt.Errorf("random service.GetFacets movie count: %w", err)
	}

	if typeFilter != TypeMovie {
		dramaGenres, err = s.dramaRepo.GetPlannedGenres(ctx, profileID)
		if err != nil {
			return nil, fmt.Errorf("random service.GetFacets drama genres: %w", err)
		}
	}
	if typeFilter != TypeSeries {
		movieGenres, err = s.movieRepo.GetPlannedGenres(ctx, profileID)
		if err != nil {
			return nil, fmt.Errorf("random service.GetFacets movie genres: %w", err)
		}
	}

	return &FacetsOutput{
		Genres:    mergeUniqueSorted(dramaGenres, movieGenres),
		HasMovies: movieCount > 0,
		HasSeries: dramaCount > 0,
	}, nil
}

// PickOutput — ответ GET /api/v1/random/pick. Форма ответа уже готова к отрисовке
// на фронте (без пересчётов на клиенте): rating уже переведён в шкалу 0-5, genres —
// массив (даже когда жанр у тайтла один), country — код как в БД, без обрезки.
type PickOutput struct {
	ID                 int64    `json:"id"`
	Kind               Kind     `json:"kind"`
	Title              string   `json:"title"`
	Year               int      `json:"year"`
	Genres             []string `json:"genres"`
	Country            string   `json:"country,omitempty"`
	Cover              string   `json:"cover,omitempty"`
	WatchURL           string   `json:"watch_url,omitempty"`
	Rating             *int     `json:"rating,omitempty"`         // шкала 0-5, уже округлено из 0-10
	EpisodesTotal      int      `json:"episodes_total,omitempty"` // сумма episode_count по всем сезонам
	EpisodeDurationMin *int     `json:"episode_duration_min,omitempty"`
	Seasons            int      `json:"seasons,omitempty"` // количество сезонов
}

// Pick подбирает один случайный тайтл в статусе "planned" под фильтры.
//
// excludeKind/excludeID — id и тип предыдущего результата (для "Ещё раз"), чтобы не
// выпадал тот же тайтл подряд. Если после исключения пул под эти фильтры опустел
// (был единственный тайтл) — откатываемся и возвращаем его же: лучше повторить,
// чем показать пустой экран там, где явно есть на что рандомить.
func (s *Service) Pick(ctx context.Context, profileID int64, typeFilter Type, genre string, excludeKind Kind, excludeID int64) (*PickOutput, error) {
	var dramaCount, movieCount int
	var err error

	if typeFilter != TypeMovie {
		dramaCount, err = s.dramaRepo.CountPlanned(ctx, profileID, genre)
		if err != nil {
			return nil, fmt.Errorf("random service.Pick drama count: %w", err)
		}
	}
	if typeFilter != TypeSeries {
		movieCount, err = s.movieRepo.CountPlanned(ctx, profileID, genre)
		if err != nil {
			return nil, fmt.Errorf("random service.Pick movie count: %w", err)
		}
	}

	total := dramaCount + movieCount
	if total == 0 {
		return nil, ErrNoMatch
	}

	pickDrama := false
	switch typeFilter {
	case TypeSeries:
		pickDrama = true
	case TypeMovie:
		pickDrama = false
	default:
		// Взвешенный выбор: вероятность попасть в дорамы пропорциональна размеру пула дорам,
		// а не фиксированные 50/50 — иначе единственный фильм выпадал бы так же часто,
		// как один из полусотни дорам.
		pickDrama = rand.Intn(total) < dramaCount
	}

	if pickDrama {
		return s.pickDrama(ctx, profileID, genre, excludeKind, excludeID)
	}
	return s.pickMovie(ctx, profileID, genre, excludeKind, excludeID)
}

func (s *Service) pickDrama(ctx context.Context, profileID int64, genre string, excludeKind Kind, excludeID int64) (*PickOutput, error) {
	dramaExcludeID := int64(0)
	if excludeKind == KindDrama {
		dramaExcludeID = excludeID
	}

	d, err := s.dramaRepo.GetRandomPlanned(ctx, profileID, genre, dramaExcludeID)
	if err != nil {
		return nil, fmt.Errorf("random service.Pick drama: %w", err)
	}
	if d == nil && dramaExcludeID > 0 {
		// исключение съело единственный подходящий тайтл — откатываемся и берём его же
		d, err = s.dramaRepo.GetRandomPlanned(ctx, profileID, genre, 0)
		if err != nil {
			return nil, fmt.Errorf("random service.Pick drama fallback: %w", err)
		}
	}
	if d == nil {
		return nil, ErrNoMatch
	}
	return toDramaPick(d), nil
}

func (s *Service) pickMovie(ctx context.Context, profileID int64, genre string, excludeKind Kind, excludeID int64) (*PickOutput, error) {
	movieExcludeID := int64(0)
	if excludeKind == KindMovie {
		movieExcludeID = excludeID
	}

	m, err := s.movieRepo.GetRandomPlanned(ctx, profileID, genre, movieExcludeID)
	if err != nil {
		return nil, fmt.Errorf("random service.Pick movie: %w", err)
	}
	if m == nil && movieExcludeID > 0 {
		m, err = s.movieRepo.GetRandomPlanned(ctx, profileID, genre, 0)
		if err != nil {
			return nil, fmt.Errorf("random service.Pick movie fallback: %w", err)
		}
	}
	if m == nil {
		return nil, ErrNoMatch
	}
	return toMoviePick(m), nil
}

// ── mapping ──────────────────────────────────────────────────────────────────────

func toDramaPick(d *dramadomain.Drama) *PickOutput {
	totalEpisodes := 0
	for _, season := range d.Seasons() {
		totalEpisodes += season.EpisodeCount
	}

	return &PickOutput{
		ID:                 d.ID(),
		Kind:               KindDrama,
		Title:              d.Title(),
		Year:               d.ReleaseYear(),
		Genres:             genreSlice(d.Genre()),
		Country:            d.Country(),
		Cover:              d.PosterURL(),
		WatchURL:           d.WatchURL(),
		Rating:             ratingToFiveScale(d.Rating()),
		EpisodesTotal:      totalEpisodes,
		EpisodeDurationMin: d.EpisodeDurationMin(),
		Seasons:            len(d.Seasons()),
	}
}

func toMoviePick(m *moviedomain.Movie) *PickOutput {
	year := 0
	if m.ReleaseYear() != nil {
		year = *m.ReleaseYear()
	}

	return &PickOutput{
		ID:      m.ID(),
		Kind:    KindMovie,
		Title:   m.Title(),
		Year:    year,
		Genres:  genreSlice(m.Genre()),
		Country: m.Country(),
		// У фильмов сознательно нет cover/watch_url/rating/episodes — таких полей
		// нет и в самой доменной модели (см. movie.entity.go), не выдумываем их тут.
	}
}

func genreSlice(genre string) []string {
	if genre == "" {
		return []string{}
	}
	return []string{genre}
}

// ratingToFiveScale переводит рейтинг из БД (0-10, как ставит пользователь) в шкалу 0-5
// (пять звёзд на карточке) — тот же расчёт, что раньше делал фронт в adaptDramaFromApi,
// просто теперь на бэке, чтобы фронт не считал вообще ничего.
func ratingToFiveScale(rating *float64) *int {
	if rating == nil {
		return nil
	}
	v := int(math.Round(*rating / 2))
	if v < 1 {
		v = 1
	}
	return &v
}

func mergeUniqueSorted(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, v := range list {
			if v == "" || seen[v] {
				continue
			}
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}
