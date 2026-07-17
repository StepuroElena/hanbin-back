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
			profile_id, title, watch_url, release_year,
			release_tag, translation_tag, genre, rating,
			watch_status, country,
			is_archived, episode_duration_min, voiceover, poster_url, seasons, progress,
			created_at, updated_at
		) VALUES (
			$1,  $2,  $3,  $4,
			$5,  $6,  $7,  $8,
			$9,  $10,
			$11, $12, $13, $14, $15, $16,
			$17, $18
		) RETURNING id`

	var id int64
	err = r.db.QueryRowContext(ctx, q,
		d.ProfileID(),
		d.Title(),
		d.WatchURL(),
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
		SELECT id, profile_id, title, watch_url, release_year,
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
		SELECT id, profile_id, title, watch_url, release_year,
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
			release_year         = $3,
			release_tag          = $4,
			translation_tag      = $5,
			genre                = $6,
			rating               = $7,
			watch_status         = $8,
			country              = $9,
			episode_duration_min = $10,
			voiceover            = $11,
			poster_url           = $12,
			seasons              = $13,
			progress             = $14,
			updated_at           = $15
		WHERE id = $16`

	res, err := r.db.ExecContext(ctx, q,
		d.Title(),
		d.WatchURL(),
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
func (r *postgresRepository) GetStatsByProfileID(ctx context.Context, profileID int64) (*domain.Stats, error) {
	const q = `
		SELECT
			COUNT(*) FILTER (WHERE watch_status = 'completed') AS watched,
			COUNT(*) FILTER (WHERE watch_status = 'watching')  AS watching,
			COUNT(*) FILTER (WHERE watch_status = 'planned')   AS planned,
			COUNT(*) FILTER (WHERE watch_status = 'dropped')   AS dropped
		FROM dramas
		WHERE profile_id = $1 AND is_archived = false`

	var s domain.Stats
	err := r.db.QueryRowContext(ctx, q, profileID).Scan(
		&s.DramasWatched,
		&s.DramasWatching,
		&s.DramasPlanned,
		&s.DramasDropped,
	)
	if err != nil {
		return nil, fmt.Errorf("drama repository.GetStatsByProfileID: %w", err)
	}

	s.TotalEpisodes = s.DramasWatched + s.DramasWatching
	s.TotalHours = int(float64(s.TotalEpisodes) * 45 / 60)

	return &s, nil
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
		&id, &profileID, &title, &watchURL, &releaseYear,
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
