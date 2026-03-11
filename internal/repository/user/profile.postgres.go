package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

// postgresRepository — реализация domain.Repository для PostgreSQL.
type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создаёт репозиторий профилей.
func NewPostgresRepository(db *sql.DB) domain.Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, p *domain.Profile) (int64, error) {
	const q = `
		INSERT INTO profiles (user_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		p.UserID(), p.Name(), p.CreatedAt(), p.UpdatedAt(),
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, domain.ErrProfileExists
		}
		return 0, fmt.Errorf("repository.Create: %w", err)
	}
	return id, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id int64) (*domain.Profile, error) {
	const q = `SELECT id, user_id, name, created_at, updated_at FROM profiles WHERE id = $1`
	row := r.db.QueryRowContext(ctx, q, id)
	return scanProfile(row)
}

func (r *postgresRepository) GetByUserID(ctx context.Context, userID int64) (*domain.Profile, error) {
	const q = `SELECT id, user_id, name, created_at, updated_at FROM profiles WHERE user_id = $1`
	row := r.db.QueryRowContext(ctx, q, userID)
	return scanProfile(row)
}

func (r *postgresRepository) Update(ctx context.Context, p *domain.Profile) error {
	const q = `UPDATE profiles SET name = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, q, p.Name(), p.UpdatedAt(), p.ID())
	if err != nil {
		return fmt.Errorf("repository.Update: %w", err)
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM profiles WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("repository.Delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ── Вспомогательные функции ───────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProfile(row rowScanner) (*domain.Profile, error) {
	var (
		id        int64
		userID    int64
		name      string
		createdAt time.Time
		updatedAt time.Time
	)
	if err := row.Scan(&id, &userID, &name, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repository.scan: %w", err)
	}
	return domain.Reconstitute(id, userID, name, createdAt, updatedAt), nil
}

// isUniqueViolation проверяет SQLSTATE 23505 (unique_violation в PostgreSQL).
func isUniqueViolation(err error) bool {
	return err != nil && containsAny(err.Error(), "23505", "unique constraint", "duplicate key")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
