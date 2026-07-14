// Package scrapersvc реализует гибридную стратегию получения данных о драме:
// cache-aside с TTL поверх internal/scraper.
//
// Логика:
//  1. Смотрим в scrape_cache по ключу normalized(title) + host(siteURL).
//  2. Если запись есть и она свежее TTL — отдаём из кеша, на внешний сайт не ходим.
//  3. Если записи нет или она протухла — идём скрейпить сайт живьём (internal/scraper),
//     и при успехе обновляем кеш (best-effort — ошибка записи в кеш не должна
//     ронять ответ пользователю, раз данные уже получены).
//
// Почему не полное зеркалирование чужого каталога и не 100% live-скрейп на каждый
// запрос — см. обсуждение в истории проекта: полное зеркалирование требует фоновых
// воркеров и юридически спорнее, а чистый live-скрейп на каждый чих медленный и
// зависит от аптайма/антибот-защиты стороннего сайта. Cache-aside — золотая середина
// для трекера личных списков просмотра.
package scrapersvc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hanbin/hanbin-back/internal/repository/scrapecache"
	"github.com/hanbin/hanbin-back/internal/scraper"
)

// DefaultTTL — как долго закешированная запись считается свежей.
// Метаданные дорамы (год, жанры, страна) почти никогда не меняются, но статус
// выпуска/перевода и рейтинг могут обновляться, поэтому TTL не берём слишком большим.
const DefaultTTL = 14 * 24 * time.Hour

type Service struct {
	repo scrapecache.Repository
	ttl  time.Duration
}

// NewService создаёт сервис скрейпинга с TTL по умолчанию.
func NewService(repo scrapecache.Repository) *Service {
	return &Service{repo: repo, ttl: DefaultTTL}
}

// WithTTL позволяет переопределить TTL (полезно в тестах).
func (s *Service) WithTTL(ttl time.Duration) *Service {
	s.ttl = ttl
	return s
}

// Scrape возвращает данные о драме — из кеша, если он свежий, иначе идёт на сайт
// и обновляет кеш. Ошибки как у scraper.Scrape: scraper.ErrNotFound, если дорама
// не найдена ни в кеше, ни на сайте.
func (s *Service) Scrape(ctx context.Context, title, siteURL string) (*scraper.DramaInfo, error) {
	key := cacheKey(title, siteURL)

	if s.repo != nil {
		if entry, err := s.repo.Get(ctx, key); err == nil {
			if time.Since(entry.ScrapedAt) < s.ttl {
				var info scraper.DramaInfo
				if jsonErr := json.Unmarshal(entry.Data, &info); jsonErr == nil {
					return &info, nil
				}
				// Битые данные в кеше — игнорируем и идём скрейпить заново,
				// не роняем запрос из-за повреждённой записи.
			}
			// Запись есть, но протухла — идём обновлять её живым скрейпом ниже.
		}
		// err == scrapecache.ErrNotFound или любая другая ошибка чтения кеша —
		// не блокируем пользователя, просто идём скрейпить сайт напрямую.
	}

	info, err := scraper.Scrape(ctx, title, siteURL)
	if err != nil {
		// Отрицательные результаты (не найдено) сознательно не кешируем:
		// дорама может появиться на сайте позже, а хранить "миссы" — лишняя сложность
		// ради небольшой экономии на редком случае повторного поиска того же названия.
		return nil, err
	}

	if s.repo != nil {
		if data, jsonErr := json.Marshal(info); jsonErr == nil {
			// Best-effort: ошибка записи в кеш не должна ломать успешный ответ.
			_ = s.repo.Upsert(ctx, key, title, siteURL, data)
		}
	}

	return info, nil
}

// cacheKey строит стабильный ключ кеша: normalized(title) + "|" + host сайта.
// Это гарантирует, что один и тот же тайтл на одном и том же сайте не скрейпится
// заново каждым отдельным пользователем — а разные сайты для одного тайтла
// кешируются отдельно, т.к. могут отдавать разные постеры/год/статус перевода.
func cacheKey(title, siteURL string) string {
	host := siteURL
	if u, err := url.Parse(siteURL); err == nil && u.Host != "" {
		host = u.Host
	}
	host = strings.ToLower(strings.TrimPrefix(host, "www."))

	normTitle := strings.ToLower(strings.TrimSpace(title))
	normTitle = strings.Join(strings.Fields(normTitle), " ") // схлопываем лишние пробелы

	return fmt.Sprintf("%s|%s", normTitle, host)
}
