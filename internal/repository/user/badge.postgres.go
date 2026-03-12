package user

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

// postgresBadgeRepository — реализация domain.BadgeRepository.
type postgresBadgeRepository struct {
	db *sql.DB
}

// NewPostgresBadgeRepository создаёт репозиторий бэйджей.
func NewPostgresBadgeRepository(db *sql.DB) domain.BadgeRepository {
	return &postgresBadgeRepository{db: db}
}

// GetByUserID возвращает все бэйджи пользователя.
func (r *postgresBadgeRepository) GetByUserID(ctx context.Context, userID int64) ([]*domain.Badge, error) {
	const q = `
		SELECT id, user_id, code, name,
		       COALESCE(description, ''),
		       COALESCE(icon, ''),
		       earned_at
		FROM badges
		WHERE user_id = $1
		ORDER BY earned_at ASC`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("badgeRepository.GetByUserID: %w", err)
	}
	defer rows.Close()

	var badges []*domain.Badge
	for rows.Next() {
		var (
			id          int64
			uid         int64
			code        string
			name        string
			description string
			icon        string
			earnedAt    time.Time
		)
		if err := rows.Scan(&id, &uid, &code, &name, &description, &icon, &earnedAt); err != nil {
			return nil, fmt.Errorf("badgeRepository.GetByUserID scan: %w", err)
		}
		badges = append(badges, domain.ReconstituteBadge(id, uid, code, name, description, icon, earnedAt))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("badgeRepository.GetByUserID rows: %w", err)
	}
	return badges, nil
}
