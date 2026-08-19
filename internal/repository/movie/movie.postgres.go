package movie

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/movie"
)

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создаёт репозиторий фильмов для PostgreSQL.
func NewPostgresRepository(db *sql.DB) domain.Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, m *domain.Movie) (int64, error) {
	const q = `
		INSERT INTO movies (profile_id, title, genre, country, category, release_year, watch_status, is_archived, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		m.ProfileID(),
		m.Title(),
		m.Genre(),
		m.Country(),
		m.Category(),
		m.ReleaseYear(),
		string(m.WatchStatus()),
		m.IsArchived(),
		m.CreatedAt(),
		m.UpdatedAt(),
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("movie repository.Create: %w", err)
	}
	return id, nil
}

func (r *postgresRepository) GetAllByProfileID(ctx context.Context, profileID int64) ([]*domain.Movie, error) {
	const q = `
		SELECT id, profile_id, title, genre, country, category, release_year, watch_status, is_archived, created_at, updated_at
		FROM movies
		WHERE profile_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, profileID)
	if err != nil {
		return nil, fmt.Errorf("movie repository.GetAllByProfileID: %w", err)
	}
	defer rows.Close()

	var movies []*domain.Movie
	for rows.Next() {
		m, err := scanMovie(rows)
		if err != nil {
			return nil, err
		}
		movies = append(movies, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("movie repository.GetAllByProfileID rows: %w", err)
	}
	return movies, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id int64) (*domain.Movie, error) {
	const q = `
		SELECT id, profile_id, title, genre, country, category, release_year, watch_status, is_archived, created_at, updated_at
		FROM movies
		WHERE id = $1`

	return scanMovie(r.db.QueryRowContext(ctx, q, id))
}

func (r *postgresRepository) UpdateWatchStatus(ctx context.Context, id int64, status domain.WatchStatus) error {
	const q = `
		UPDATE movies
		SET watch_status = $1, updated_at = $2
		WHERE id = $3`

	res, err := r.db.ExecContext(ctx, q, string(status), time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("movie repository.UpdateWatchStatus: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("movie repository.UpdateWatchStatus rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepository) UpdateArchived(ctx context.Context, id int64, isArchived bool) error {
	const q = `
		UPDATE movies
		SET is_archived = $1, updated_at = $2
		WHERE id = $3`

	res, err := r.db.ExecContext(ctx, q, isArchived, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("movie repository.UpdateArchived: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("movie repository.UpdateArchived rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Update пишет все редактируемые поля фильма (название/жанр/страна/категория/год/статус) одним UPDATE.
// is_archived тут сознательно не трогаем — архив меняется только через отдельный UpdateArchived выше.
func (r *postgresRepository) Update(ctx context.Context, m *domain.Movie) error {
	const q = `
		UPDATE movies SET
			title        = $1,
			genre        = $2,
			country      = $3,
			category     = $4,
			release_year = $5,
			watch_status = $6,
			updated_at   = $7
		WHERE id = $8`

	res, err := r.db.ExecContext(ctx, q,
		m.Title(),
		m.Genre(),
		m.Country(),
		m.Category(),
		m.ReleaseYear(),
		string(m.WatchStatus()),
		time.Now().UTC(),
		m.ID(),
	)
	if err != nil {
		return fmt.Errorf("movie repository.Update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("movie repository.Update rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM movies WHERE id = $1`

	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("movie repository.Delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("movie repository.Delete rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CountPlanned считает количество неархивированных фильмов в статусе "planned", опционально под жанр.
func (r *postgresRepository) CountPlanned(ctx context.Context, profileID int64, genre string) (int, error) {
	const q = `
		SELECT COUNT(*) FROM movies
		WHERE profile_id = $1 AND watch_status = 'planned' AND is_archived = false
		  AND ($2 = '' OR genre = $2)`

	var count int
	if err := r.db.QueryRowContext(ctx, q, profileID, genre).Scan(&count); err != nil {
		return 0, fmt.Errorf("movie repository.CountPlanned: %w", err)
	}
	return count, nil
}

// GetRandomPlanned возвращает один случайный неархивированный фильм в статусе "planned".
func (r *postgresRepository) GetRandomPlanned(ctx context.Context, profileID int64, genre string, excludeID int64) (*domain.Movie, error) {
	const q = `
		SELECT id, profile_id, title, genre, country, category, release_year, watch_status, is_archived, created_at, updated_at
		FROM movies
		WHERE profile_id = $1 AND watch_status = 'planned' AND is_archived = false
		  AND ($2 = '' OR genre = $2)
		  AND ($3 <= 0 OR id <> $3)
		ORDER BY random()
		LIMIT 1`

	m, err := scanMovie(r.db.QueryRowContext(ctx, q, profileID, genre, excludeID))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil // пустой пул для этих фильтров — ожидаемый исход, не ошибка
		}
		return nil, fmt.Errorf("movie repository.GetRandomPlanned: %w", err)
	}
	return m, nil
}

// GetPlannedGenres возвращает жанры, реально присутствующие среди фильмов в статусе "planned".
func (r *postgresRepository) GetPlannedGenres(ctx context.Context, profileID int64) ([]string, error) {
	const q = `
		SELECT DISTINCT genre FROM movies
		WHERE profile_id = $1 AND watch_status = 'planned' AND is_archived = false AND genre <> ''
		ORDER BY genre`

	rows, err := r.db.QueryContext(ctx, q, profileID)
	if err != nil {
		return nil, fmt.Errorf("movie repository.GetPlannedGenres: %w", err)
	}
	defer rows.Close()

	result := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("movie repository.GetPlannedGenres scan: %w", err)
		}
		result = append(result, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("movie repository.GetPlannedGenres rows: %w", err)
	}
	return result, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMovie(row rowScanner) (*domain.Movie, error) {
	var (
		id          int64
		profileID   int64
		title       string
		genre       string
		country     string
		category    string
		releaseYear sql.NullInt32
		watchStatus string
		isArchived  bool
		createdAt   time.Time
		updatedAt   time.Time
	)

	if err := row.Scan(&id, &profileID, &title, &genre, &country, &category, &releaseYear, &watchStatus, &isArchived, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("movie repository.scan: %w", err)
	}

	var yearPtr *int
	if releaseYear.Valid {
		v := int(releaseYear.Int32)
		yearPtr = &v
	}

	return domain.Reconstitute(id, profileID, title, genre, country, category, yearPtr, domain.WatchStatus(watchStatus), isArchived, createdAt, updatedAt), nil
}
