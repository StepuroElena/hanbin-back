package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

// postgresUserRepository — реализация domain.UserRepository для PostgreSQL.
type postgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository создаёт репозиторий пользователей.
func NewPostgresUserRepository(db *sql.DB) domain.UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(ctx context.Context, u *domain.User) (int64, error) {
	const q = `
		INSERT INTO users (email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var id int64
	err := r.db.QueryRowContext(ctx, q,
		u.Email(), u.PasswordHash(), u.CreatedAt(), u.UpdatedAt(),
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, domain.ErrEmailNotUnique
		}
		return 0, fmt.Errorf("userRepository.Create: %w", err)
	}
	return id, nil
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	const q = `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, q, id)
	return scanUser(row)
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = $1`
	row := r.db.QueryRowContext(ctx, q, email)
	return scanUser(row)
}

// ── Вспомогательные функции ───────────────────────────────────────────────────

func scanUser(row rowScanner) (*domain.User, error) {
	var (
		id           int64
		email        string
		passwordHash string
		createdAt    time.Time
		updatedAt    time.Time
	)
	if err := row.Scan(&id, &email, &passwordHash, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("userRepository.scan: %w", err)
	}
	return domain.ReconstituteUser(id, email, passwordHash, createdAt, updatedAt), nil
}
