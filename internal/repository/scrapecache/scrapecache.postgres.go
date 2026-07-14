// Package scrapecache — персистентность для cache-aside кеша скрейпинга.
// Хранит сырые результаты scraper.DramaInfo как JSONB, ключ — normalized
// title + host сайта-источника. TTL-логика (свежая запись или нет) живёт
// на уровне сервиса (internal/service/scraper), а не здесь — репозиторий
// только читает/пишет и не принимает решений о валидности данных.
package scrapecache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound — в кеше нет записи с таким ключом.
var ErrNotFound = errors.New("scrape cache: entry not found")

// Entry — закешированная запись.
type Entry struct {
	Data      []byte    // сериализованный JSON scraper.DramaInfo
	ScrapedAt time.Time // когда запись была сохранена/обновлена
}

// Repository — интерфейс персистентности кеша скрейпинга.
type Repository interface {
	// Get возвращает закешированную запись по ключу.
	// Возвращает ErrNotFound, если записи нет — это НЕ ошибка уровня приложения,
	// вызывающий код просто идёт скрейпить заново.
	Get(ctx context.Context, cacheKey string) (*Entry, error)

	// Upsert сохраняет/обновляет запись кеша. Best-effort операция:
	// вызывающий код (сервис) не должен падать, если запись в кеш не сохранилась.
	Upsert(ctx context.Context, cacheKey, title, siteURL string, data []byte) error
}

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository создаёт репозиторий кеша скрейпинга для PostgreSQL.
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Get(ctx context.Context, cacheKey string) (*Entry, error) {
	const q = `SELECT data, scraped_at FROM scrape_cache WHERE cache_key = $1`

	var e Entry
	err := r.db.QueryRowContext(ctx, q, cacheKey).Scan(&e.Data, &e.ScrapedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scrapecache repository.Get: %w", err)
	}
	return &e, nil
}

func (r *postgresRepository) Upsert(ctx context.Context, cacheKey, title, siteURL string, data []byte) error {
	const q = `
		INSERT INTO scrape_cache (cache_key, title, site_url, data, scraped_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (cache_key) DO UPDATE SET
			title      = EXCLUDED.title,
			site_url   = EXCLUDED.site_url,
			data       = EXCLUDED.data,
			scraped_at = NOW()`

	if _, err := r.db.ExecContext(ctx, q, cacheKey, title, siteURL, data); err != nil {
		return fmt.Errorf("scrapecache repository.Upsert: %w", err)
	}
	return nil
}
