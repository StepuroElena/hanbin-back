package moviecategory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/moviecategory"
)

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создаёт репозиторий категорий фильмов для PostgreSQL.
func NewPostgresRepository(db *sql.DB) domain.Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, c *domain.MovieCategory) (int64, error) {
	const q = `
		INSERT INTO movie_categories (profile_id, name, position, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		c.ProfileID(), c.Name(), c.Position(), c.Enabled(), c.CreatedAt(), c.UpdatedAt(),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("moviecategory repository.Create: %w", err)
	}
	return id, nil
}

func (r *postgresRepository) GetAllByProfileID(ctx context.Context, profileID int64) ([]*domain.MovieCategory, error) {
	const q = `
		SELECT id, profile_id, name, position, enabled, created_at, updated_at
		FROM movie_categories
		WHERE profile_id = $1
		ORDER BY position ASC, id ASC`

	rows, err := r.db.QueryContext(ctx, q, profileID)
	if err != nil {
		return nil, fmt.Errorf("moviecategory repository.GetAllByProfileID: %w", err)
	}
	defer rows.Close()

	var categories []*domain.MovieCategory
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("moviecategory repository.GetAllByProfileID rows: %w", err)
	}
	return categories, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id int64) (*domain.MovieCategory, error) {
	const q = `
		SELECT id, profile_id, name, position, enabled, created_at, updated_at
		FROM movie_categories
		WHERE id = $1`

	return scanCategory(r.db.QueryRowContext(ctx, q, id))
}

func (r *postgresRepository) Update(ctx context.Context, c *domain.MovieCategory) error {
	const q = `
		UPDATE movie_categories SET
			name       = $1,
			position   = $2,
			enabled    = $3,
			updated_at = $4
		WHERE id = $5`

	res, err := r.db.ExecContext(ctx, q, c.Name(), c.Position(), c.Enabled(), time.Now().UTC(), c.ID())
	if err != nil {
		return fmt.Errorf("moviecategory repository.Update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("moviecategory repository.Update rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM movie_categories WHERE id = $1`

	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("moviecategory repository.Delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("moviecategory repository.Delete rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepository) CountByProfileID(ctx context.Context, profileID int64) (int, error) {
	const q = `SELECT COUNT(*) FROM movie_categories WHERE profile_id = $1`

	var count int
	if err := r.db.QueryRowContext(ctx, q, profileID).Scan(&count); err != nil {
		return 0, fmt.Errorf("moviecategory repository.CountByProfileID: %w", err)
	}
	return count, nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCategory(row rowScanner) (*domain.MovieCategory, error) {
	var (
		id        int64
		profileID int64
		name      string
		position  int
		enabled   bool
		createdAt time.Time
		updatedAt time.Time
	)

	if err := row.Scan(&id, &profileID, &name, &position, &enabled, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("moviecategory repository.scan: %w", err)
	}

	return domain.Reconstitute(id, profileID, name, position, enabled, createdAt, updatedAt), nil
}
