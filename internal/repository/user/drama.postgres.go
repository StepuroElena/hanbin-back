package user

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

// postgresDramaRepository — реализация domain.DramaRepository.
type postgresDramaRepository struct {
	db *sql.DB
}

// NewPostgresDramaRepository создаёт репозиторий дорам.
func NewPostgresDramaRepository(db *sql.DB) domain.DramaRepository {
	return &postgresDramaRepository{db: db}
}

// GetByUserID возвращает все дорамы пользователя вместе с тегами.
func (r *postgresDramaRepository) GetByUserID(ctx context.Context, userID int64) ([]*domain.Drama, error) {
	const q = `
		SELECT id, user_id, name,
		       COALESCE(year, 0),
		       COALESCE(genre, ''),
		       COALESCE(country, ''),
		       COALESCE(doramatv_rating, 0),
		       watch_status,
		       current_episode,
		       COALESCE(total_episodes, 0),
		       COALESCE(doramatv_url, ''),
		       created_at, updated_at
		FROM dramas
		WHERE user_id = $1
		ORDER BY updated_at DESC`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("dramaRepository.GetByUserID: %w", err)
	}
	defer rows.Close()

	var dramas []*domain.Drama
	var ids []int64

	for rows.Next() {
		var (
			id             int64
			uid            int64
			name           string
			year           int
			genre          string
			country        string
			doramatvRating float64
			watchStatus    string
			currentEp      int
			totalEp        int
			doramatvURL    string
			createdAt      time.Time
			updatedAt      time.Time
		)
		if err := rows.Scan(&id, &uid, &name, &year, &genre, &country,
			&doramatvRating, &watchStatus, &currentEp, &totalEp,
			&doramatvURL, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("dramaRepository.GetByUserID scan: %w", err)
		}

		d := domain.ReconstituteDrama(
			id, uid, name, year, genre, country,
			doramatvRating, domain.WatchStatus(watchStatus),
			currentEp, totalEp, doramatvURL,
			nil, // теги добавим ниже
			createdAt, updatedAt,
		)
		dramas = append(dramas, d)
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dramaRepository.GetByUserID rows: %w", err)
	}

	if len(dramas) == 0 {
		return dramas, nil
	}

	// Подтягиваем теги одним запросом для всех дорам.
	tagsMap, err := r.fetchTags(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Переприсваиваем с тегами.
	for i, d := range dramas {
		tags := tagsMap[d.ID()]
		dramas[i] = domain.ReconstituteDrama(
			d.ID(), d.UserID(), d.Name(), d.Year(),
			d.Genre(), d.Country(), d.DoramatvRating(),
			d.WatchStatus(), d.CurrentEpisode(), d.TotalEpisodes(),
			d.DoramatvURL(), tags,
			d.CreatedAt(), d.UpdatedAt(),
		)
	}

	return dramas, nil
}

// fetchTags возвращает теги для списка ID дорам в виде map[dramaID][]tag.
func (r *postgresDramaRepository) fetchTags(ctx context.Context, ids []int64) (map[int64][]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Строим запрос с inline-плейсхолдерами: WHERE drama_id IN ($1, $2, $3, ...)
	// Это единственный способ без pq.Array и без unnest.
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf(
		"SELECT drama_id, tag FROM drama_tags WHERE drama_id IN (%s)",
		strings.Join(placeholders, ","),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("dramaRepository.fetchTags: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]string)
	for rows.Next() {
		var dramaID int64
		var tag string
		if err := rows.Scan(&dramaID, &tag); err != nil {
			return nil, fmt.Errorf("dramaRepository.fetchTags scan: %w", err)
		}
		result[dramaID] = append(result[dramaID], tag)
	}
	return result, rows.Err()
}


