package streamingsite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/streamingsite"
)

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создаёт репозиторий сайтов просмотра для PostgreSQL.
func NewPostgresRepository(db *sql.DB) domain.Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, s *domain.StreamingSite) (int64, error) {
	const q = `
		INSERT INTO streaming_sites (profile_id, name, url, language, position, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		s.ProfileID(), s.Name(), s.URL(), string(s.Language()), s.Position(), s.Enabled(), s.CreatedAt(), s.UpdatedAt(),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("streamingsite repository.Create: %w", err)
	}
	return id, nil
}

func (r *postgresRepository) GetAllByProfileID(ctx context.Context, profileID int64) ([]*domain.StreamingSite, error) {
	const q = `
		SELECT id, profile_id, name, url, language, position, enabled, created_at, updated_at
		FROM streaming_sites
		WHERE profile_id = $1
		ORDER BY position ASC, id ASC`

	rows, err := r.db.QueryContext(ctx, q, profileID)
	if err != nil {
		return nil, fmt.Errorf("streamingsite repository.GetAllByProfileID: %w", err)
	}
	defer rows.Close()

	var sites []*domain.StreamingSite
	for rows.Next() {
		s, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		sites = append(sites, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("streamingsite repository.GetAllByProfileID rows: %w", err)
	}
	return sites, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id int64) (*domain.StreamingSite, error) {
	const q = `
		SELECT id, profile_id, name, url, language, position, enabled, created_at, updated_at
		FROM streaming_sites
		WHERE id = $1`

	return scanSite(r.db.QueryRowContext(ctx, q, id))
}

func (r *postgresRepository) Update(ctx context.Context, s *domain.StreamingSite) error {
	const q = `
		UPDATE streaming_sites SET
			name       = $1,
			url        = $2,
			language   = $3,
			position   = $4,
			enabled    = $5,
			updated_at = $6
		WHERE id = $7`

	res, err := r.db.ExecContext(ctx, q, s.Name(), s.URL(), string(s.Language()), s.Position(), s.Enabled(), time.Now().UTC(), s.ID())
	if err != nil {
		return fmt.Errorf("streamingsite repository.Update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("streamingsite repository.Update rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM streaming_sites WHERE id = $1`

	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("streamingsite repository.Delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("streamingsite repository.Delete rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *postgresRepository) CountByProfileID(ctx context.Context, profileID int64) (int, error) {
	const q = `SELECT COUNT(*) FROM streaming_sites WHERE profile_id = $1`

	var count int
	if err := r.db.QueryRowContext(ctx, q, profileID).Scan(&count); err != nil {
		return 0, fmt.Errorf("streamingsite repository.CountByProfileID: %w", err)
	}
	return count, nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSite(row rowScanner) (*domain.StreamingSite, error) {
	var (
		id        int64
		profileID int64
		name      string
		url       string
		languageS string
		position  int
		enabled   bool
		createdAt time.Time
		updatedAt time.Time
	)

	if err := row.Scan(&id, &profileID, &name, &url, &languageS, &position, &enabled, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("streamingsite repository.scan: %w", err)
	}

	return domain.Reconstitute(id, profileID, name, url, domain.Language(languageS), position, enabled, createdAt, updatedAt), nil
}
