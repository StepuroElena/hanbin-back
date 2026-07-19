package drama

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/drama"
)

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создаёт репозиторий драм для PostgreSQL.
func NewPostgresRepository(db *sql.DB) domain.Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, d *domain.Drama) (int64, error) {
	seasonsJSON, err := json.Marshal(d.Seasons())
	if err != nil {
		return 0, fmt.Errorf("drama repository.Create marshal seasons: %w", err)
	}
	progressJSON, err := json.Marshal(d.Progress())
	if err != nil {
		return 0, fmt.Errorf("drama repository.Create marshal progress: %w", err)
	}

	const q = `
		INSERT INTO dramas (
			profile_id, title, watch_url, source_url, release_year,
			release_tag, translation_tag, genre, rating,
			watch_status, country,
			is_archived, episode_duration_min, voiceover, poster_url, seasons, progress,
			created_at, updated_at
		) VALUES (
			$1,  $2,  $3,  $4,  $5,
			$6,  $7,  $8,  $9,
			$10, $11,
			$12, $13, $14, $15, $16, $17,
			$18, $19
		) RETURNING id`

	var id int64
	err = r.db.QueryRowContext(ctx, q,
		d.ProfileID(),
		d.Title(),
		d.WatchURL(),
		d.SourceURL(),
		d.ReleaseYear(),
		string(d.ReleaseTag()),
		string(d.TranslationTag()),
		d.Genre(),
		d.Rating(),
		string(d.WatchStatus()),
		d.Country(),
		d.IsArchived(),
		d.EpisodeDurationMin(),
		d.Voiceover(),
		d.PosterURL(),
		seasonsJSON,
		progressJSON,
		d.CreatedAt(),
		d.UpdatedAt(),
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("drama repository.Create: %w", err)
	}
	return id, nil
}

func (r *postgresRepository) GetAllByProfileID(ctx context.Context, profileID int64) ([]*domain.Drama, error) {
	const q = `
		SELECT id, profile_id, title, watch_url, source_url, release_year,
		       release_tag, translation_tag, genre, rating,
		       watch_status, country,
		       is_archived, episode_duration_min, voiceover, poster_url, seasons, progress,
		       created_at, updated_at
		FROM dramas
		WHERE profile_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, profileID)
	if err != nil {
		return nil, fmt.Errorf("drama repository.GetAllByProfileID: %w", err)
	}
	defer rows.Close()

	var dramas []*domain.Drama
	for rows.Next() {
		d, err := scanDrama(rows)
		if err != nil {
			return nil, err
		}
		dramas = append(dramas, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("drama repository.GetAllByProfileID rows: %w", err)
	}
	return dramas, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id int64) (*domain.Drama, error) {
	const q = `
		SELECT id, profile_id, title, watch_url, source_url, release_year,
		       release_tag, translation_tag, genre, rating,
		       watch_status, country,
		       is_archived, episode_duration_min, voiceover, poster_url, seasons, progress,
		       created_at, updated_at
		FROM dramas
		WHERE id = $1`

	return scanDrama(r.db.QueryRowContext(ctx, q, id))
}

func (r *postgresRepository) UpdateArchived(ctx context.Context, id int64, isArchived bool) error {
	const q = `
		UPDATE dramas
		SET is_archived = $1, updated_at = $2
		WHERE id = $3`

	res, err := r.db.ExecContext(ctx, q, isArchived, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("drama repository.UpdateArchived: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("drama repository.UpdateArchived rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Update(ctx context.Context, d *domain.Drama) error {
	seasonsJSON, err := json.Marshal(d.Seasons())
	if err != nil {
		return fmt.Errorf("drama repository.Update marshal seasons: %w", err)
	}
	progressJSON, err := json.Marshal(d.Progress())
	if err != nil {
		return fmt.Errorf("drama repository.Update marshal progress: %w", err)
	}

	const q = `
		UPDATE dramas SET
			title                = $1,
			watch_url            = $2,
			source_url           = $3,
			release_year         = $4,
			release_tag          = $5,
			translation_tag      = $6,
			genre                = $7,
			rating               = $8,
			watch_status         = $9,
			country              = $10,
			episode_duration_min = $11,
			voiceover            = $12,
			poster_url           = $13,
			seasons              = $14,
			progress             = $15,
			updated_at           = $16
		WHERE id = $17`

	res, err := r.db.ExecContext(ctx, q,
		d.Title(),
		d.WatchURL(),
		d.SourceURL(),
		d.ReleaseYear(),
		string(d.ReleaseTag()),
		string(d.TranslationTag()),
		d.Genre(),
		d.Rating(),
		string(d.WatchStatus()),
		d.Country(),
		d.EpisodeDurationMin(),
		d.Voiceover(),
		d.PosterURL(),
		seasonsJSON,
		progressJSON,
		time.Now().UTC(),
		d.ID(),
	)
	if err != nil {
		return fmt.Errorf("drama repository.Update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("drama repository.Update rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM dramas WHERE id = $1`

	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("drama repository.Delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("drama repository.Delete rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetStatsByProfileID считает агрегированную статистику по статусам одним запросом с COUNT...FILTER,
// без выгрузки строк и без вычислений на стороне приложения. Архивированные дорамы исключены.
//
// TotalHours теперь считается не приближённо (episodes*45мин), а точно: для каждой дорамы в статусе
// «completed» берём episode_duration_min * (сумма episode_count по всем сезонам из seasons JSONB),
// суммируем по всем дорамам и переводим в часы. Дорамы без указанной длительности/сезонов
// просто не дают вклад (COALESCE в 0), а не ломают всю вычисление.
func (r *postgresRepository) GetStatsByProfileID(ctx context.Context, profileID int64) (*domain.Stats, error) {
	const q = `
		SELECT
			COUNT(*) FILTER (WHERE watch_status = 'completed') AS watched,
			COUNT(*) FILTER (WHERE watch_status = 'watching')  AS watching,
			COUNT(*) FILTER (WHERE watch_status = 'planned')   AS planned,
			COUNT(*) FILTER (WHERE watch_status = 'dropped')   AS dropped,
			COALESCE(SUM(
				CASE WHEN watch_status = 'completed' THEN
					COALESCE(episode_duration_min, 0) * COALESCE((
						SELECT SUM(COALESCE((season->>'episode_count')::int, 0))
						FROM jsonb_array_elements(COALESCE(seasons, '[]'::jsonb)) AS season
					), 0)
				ELSE 0 END
			), 0) AS total_minutes_watched
		FROM dramas
		WHERE profile_id = $1 AND is_archived = false`

	var s domain.Stats
	var totalMinutesWatched int64
	err := r.db.QueryRowContext(ctx, q, profileID).Scan(
		&s.DramasWatched,
		&s.DramasWatching,
		&s.DramasPlanned,
		&s.DramasDropped,
		&totalMinutesWatched,
	)
	if err != nil {
		return nil, fmt.Errorf("drama repository.GetStatsByProfileID: %w", err)
	}

	s.TotalEpisodes = s.DramasWatched + s.DramasWatching
	s.TotalHours = int(totalMinutesWatched / 60)

	return &s, nil
}

// GetFacetsByProfileID возвращает реально используемые страны и жанры — два DISTINCT-запроса,
// чтобы фильтры на главной показывали только то, что реально есть у пользователя.
func (r *postgresRepository) GetFacetsByProfileID(ctx context.Context, profileID int64) (*domain.Facets, error) {
	const countriesQ = `
		SELECT DISTINCT country FROM dramas
		WHERE profile_id = $1 AND is_archived = false AND country <> ''
		ORDER BY country`
	const genresQ = `
		SELECT DISTINCT genre FROM dramas
		WHERE profile_id = $1 AND is_archived = false AND genre <> ''
		ORDER BY genre`

	countries, err := r.queryDistinctStrings(ctx, countriesQ, profileID)
	if err != nil {
		return nil, fmt.Errorf("drama repository.GetFacetsByProfileID countries: %w", err)
	}
	genres, err := r.queryDistinctStrings(ctx, genresQ, profileID)
	if err != nil {
		return nil, fmt.Errorf("drama repository.GetFacetsByProfileID genres: %w", err)
	}

	return &domain.Facets{Countries: countries, Genres: genres}, nil
}

func (r *postgresRepository) queryDistinctStrings(ctx context.Context, query string, profileID int64) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// ── helpers ─────────────────────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDrama(row rowScanner) (*domain.Drama, error) {
	var (
		id                 int64
		profileID          int64
		title              string
		watchURL           string
		sourceURL          sql.NullString
		releaseYear        int
		releaseTagStr      string
		translTagStr       string
		genre              string
		rating             sql.NullFloat64
		watchStatusStr     string
		country            string
		isArchived         bool
		episodeDurationMin sql.NullInt32
		voiceover          sql.NullString
		posterURL          sql.NullString
		seasonsJSON        []byte
		progressJSON       []byte
		createdAt          time.Time
		updatedAt          time.Time
	)

	if err := row.Scan(
		&id, &profileID, &title, &watchURL, &sourceURL, &releaseYear,
		&releaseTagStr, &translTagStr, &genre, &rating,
		&watchStatusStr, &country,
		&isArchived, &episodeDurationMin, &voiceover, &posterURL, &seasonsJSON, &progressJSON,
		&createdAt, &updatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("drama repository.scan: %w", err)
	}

	var ratingPtr *float64
	if rating.Valid {
		v := rating.Float64
		ratingPtr = &v
	}

	var durationPtr *int
	if episodeDurationMin.Valid {
		v := int(episodeDurationMin.Int32)
		durationPtr = &v
	}

	var seasons []domain.Season
	if len(seasonsJSON) > 0 {
		if err := json.Unmarshal(seasonsJSON, &seasons); err != nil {
			return nil, fmt.Errorf("drama repository.scan unmarshal seasons: %w", err)
		}
	}

	var progress domain.Progress
	if len(progressJSON) > 0 {
		if err := json.Unmarshal(progressJSON, &progress); err != nil {
			return nil, fmt.Errorf("drama repository.scan unmarshal progress: %w", err)
		}
	}

	return domain.Reconstitute(
		id, profileID,
		title, watchURL,
		sourceURL.String,
		releaseYear,
		domain.ReleaseTag(releaseTagStr),
		domain.TranslationTag(translTagStr),
		genre,
		ratingPtr,
		domain.WatchStatus(watchStatusStr),
		country,
		isArchived,
		durationPtr,
		voiceover.String,
		posterURL.String,
		seasons,
		progress,
		createdAt, updatedAt,
	), nil
}
