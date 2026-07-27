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
		INSERT INTO movies (profile_id, title, genre, country, release_year, watch_status, is_archived, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		m.ProfileID(),
		m.Title(),
		m.Genre(),
		m.Country(),
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
		SELECT id, profile_id, title, genre, country, release_year, watch_status, is_archived, created_at, updated_at
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
		SELECT id, profile_id, title, genre, country, release_year, watch_status, is_archived, created_at, updated_at
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
		releaseYear sql.NullInt32
		watchStatus string
		isArchived  bool
		createdAt   time.Time
		updatedAt   time.Time
	)

	if err := row.Scan(&id, &profileID, &title, &genre, &country, &releaseYear, &watchStatus, &isArchived, &createdAt, &updatedAt); err != nil {
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

	return domain.Reconstitute(id, profileID, title, genre, country, yearPtr, domain.WatchStatus(watchStatus), isArchived, createdAt, updatedAt), nil
}
