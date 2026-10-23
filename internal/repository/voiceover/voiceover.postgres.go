package voiceover

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/voiceover"
)

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создаёт репозиторий озвучек для PostgreSQL.
func NewPostgresRepository(db *sql.DB) domain.Repository {
	return &postgresRepository{db: db}
}

// Общая часть SELECT: озвучка + персональный тогл профиля ($1). Нет строки в voiceover_settings = включена.
const selectForProfile = `
	SELECT v.id, COALESCE(v.profile_id, 0), v.name, v.position,
	       COALESCE(s.enabled, true), v.created_at, v.updated_at
	FROM voiceovers v
	LEFT JOIN voiceover_settings s ON s.voiceover_id = v.id AND s.profile_id = $1`

func (r *postgresRepository) ListForProfile(ctx context.Context, profileID int64) ([]*domain.Voiceover, error) {
	q := selectForProfile + `
	WHERE v.profile_id IS NULL OR v.profile_id = $1
	ORDER BY (v.profile_id IS NOT NULL) ASC, v.position ASC, v.id ASC`

	return r.list(ctx, q, profileID)
}

func (r *postgresRepository) ListDefaults(ctx context.Context) ([]*domain.Voiceover, error) {
	const q = `
	SELECT id, 0, name, position, true, created_at, updated_at
	FROM voiceovers
	WHERE profile_id IS NULL
	ORDER BY position ASC, id ASC`

	return r.list(ctx, q)
}

func (r *postgresRepository) GetForProfile(ctx context.Context, profileID, id int64) (*domain.Voiceover, error) {
	q := selectForProfile + `
	WHERE v.id = $2 AND (v.profile_id IS NULL OR v.profile_id = $1)`

	return scanVoiceover(r.db.QueryRowContext(ctx, q, profileID, id))
}

func (r *postgresRepository) NextCustomPosition(ctx context.Context, profileID int64) (int, error) {
	const q = `SELECT COALESCE(MAX(position) + 1, 0) FROM voiceovers WHERE profile_id = $1`

	var pos int
	if err := r.db.QueryRowContext(ctx, q, profileID).Scan(&pos); err != nil {
		return 0, fmt.Errorf("voiceover repository.NextCustomPosition: %w", err)
	}
	return pos, nil
}

func (r *postgresRepository) Create(ctx context.Context, v *domain.Voiceover) (int64, error) {
	const q = `
		INSERT INTO voiceovers (profile_id, name, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q, v.ProfileID(), v.Name(), v.Position(), v.CreatedAt(), v.UpdatedAt()).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("voiceover repository.Create: %w", err)
	}
	return id, nil
}

func (r *postgresRepository) Rename(ctx context.Context, profileID, id int64, oldName, newName string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("voiceover repository.Rename begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC()

	res, err := tx.ExecContext(ctx,
		`UPDATE voiceovers SET name = $1, updated_at = $2 WHERE id = $3 AND profile_id = $4`,
		newName, now, id, profileID)
	if err != nil {
		return fmt.Errorf("voiceover repository.Rename: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}

	// Дорамы хранят название озвучки строкой — переносим переименование и на них,
	// чтобы в таблице не осталось «осиротевших» старых названий.
	if _, err := tx.ExecContext(ctx,
		`UPDATE dramas SET voiceover = $1, updated_at = $2 WHERE profile_id = $3 AND voiceover = $4`,
		newName, now, profileID, oldName); err != nil {
		return fmt.Errorf("voiceover repository.Rename dramas: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("voiceover repository.Rename commit: %w", err)
	}
	return nil
}

func (r *postgresRepository) SetEnabled(ctx context.Context, profileID, id int64, enabled bool) error {
	const q = `
		INSERT INTO voiceover_settings (profile_id, voiceover_id, enabled, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (profile_id, voiceover_id)
		DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = now()`

	if _, err := r.db.ExecContext(ctx, q, profileID, id, enabled); err != nil {
		return fmt.Errorf("voiceover repository.SetEnabled: %w", err)
	}
	return nil
}

func (r *postgresRepository) DeleteCustom(ctx context.Context, profileID, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM voiceovers WHERE id = $1 AND profile_id = $2`, id, profileID)
	if err != nil {
		return fmt.Errorf("voiceover repository.DeleteCustom: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("voiceover repository.DeleteCustom rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

func (r *postgresRepository) list(ctx context.Context, q string, args ...any) ([]*domain.Voiceover, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("voiceover repository.list: %w", err)
	}
	defer rows.Close()

	var out []*domain.Voiceover
	for rows.Next() {
		v, err := scanVoiceover(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("voiceover repository.list rows: %w", err)
	}
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanVoiceover(row rowScanner) (*domain.Voiceover, error) {
	var (
		id, profileID        int64
		name                 string
		position             int
		enabled              bool
		createdAt, updatedAt time.Time
	)
	if err := row.Scan(&id, &profileID, &name, &position, &enabled, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("voiceover repository.scan: %w", err)
	}
	return domain.Reconstitute(id, profileID, name, position, enabled, createdAt, updatedAt), nil
}
