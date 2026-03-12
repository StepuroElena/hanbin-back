package drama

import (
	"context"
	"database/sql"
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
	const q = `
		INSERT INTO dramas (
			profile_id, title, watch_url, release_year,
			release_tag, translation_tag, genre, rating,
			watch_status, country, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12
		) RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		d.ProfileID(),
		d.Title(),
		d.WatchURL(),
		d.ReleaseYear(),
		string(d.ReleaseTag()),
		string(d.TranslationTag()),
		d.Genre(),
		d.Rating(), // *float64 — nil → NULL в PostgreSQL
		string(d.WatchStatus()),
		d.Country(),
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
		       watch_status, country, created_at, updated_at
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

// ── helpers ───────────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDrama(row rowScanner) (*domain.Drama, error) {
	var (
		id             int64
		profileID      int64
		title          string
		watchURL       string
		releaseYear    int
		releaseTagStr  string
		translTagStr   string
		genre          string
		rating         sql.NullFloat64
		watchStatusStr string
		country        string
		createdAt      time.Time
		updatedAt      time.Time
	)

	if err := row.Scan(
		&id, &profileID, &title, &watchURL, &releaseYear,
		&releaseTagStr, &translTagStr, &genre, &rating,
		&watchStatusStr, &country, &createdAt, &updatedAt,
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
		createdAt, updatedAt,
	), nil
}
