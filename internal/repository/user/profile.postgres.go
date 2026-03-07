package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

// postgresRepository — конкретная реализация domain.Repository для PostgreSQL.
// Только этот файл знает о SQL; домен и сервис об этом ничего не знают.
type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создаёт репозиторий, подключённый к переданному db.
func NewPostgresRepository(db *sql.DB) domain.Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, p *domain.Profile) (int64, error) {
	const q = `
		INSERT INTO profiles (name, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		p.Name(), p.Email(), p.CreatedAt(), p.UpdatedAt(),
	).Scan(&id)

	if err != nil {
		if isUniqueViolation(err) {
			return 0, domain.ErrEmailNotUnique
		}
		return 0, fmt.Errorf("repository.Create: %w", err)
	}
	return id, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id int64) (*domain.Profile, error) {
	const q = `SELECT id, name, email, created_at, updated_at FROM profiles WHERE id = $1`

	row := r.db.QueryRowContext(ctx, q, id)
	return scanProfile(row)
}

func (r *postgresRepository) GetByEmail(ctx context.Context, email string) (*domain.Profile, error) {
	const q = `SELECT id, name, email, created_at, updated_at FROM profiles WHERE email = $1`

	row := r.db.QueryRowContext(ctx, q, email)
	return scanProfile(row)
}

func (r *postgresRepository) Update(ctx context.Context, p *domain.Profile) error {
	const q = `
		UPDATE profiles
		SET name = $1, email = $2, updated_at = $3
		WHERE id = $4`

	_, err := r.db.ExecContext(ctx, q, p.Name(), p.Email(), p.UpdatedAt(), p.ID())
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailNotUnique
		}
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
		name      string
		email     string
		createdAt time.Time
		updatedAt time.Time
	)
	if err := row.Scan(&id, &name, &email, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("repository.scan: %w", err)
	}
	return domain.Reconstitute(id, name, email, createdAt, updatedAt), nil
}

// isUniqueViolation проверяет SQLSTATE 23505 (unique_violation в PostgreSQL).
func isUniqueViolation(err error) bool {
	return err != nil && len(err.Error()) > 0 &&
		containsAny(err.Error(), "23505", "unique constraint", "duplicate key")
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
