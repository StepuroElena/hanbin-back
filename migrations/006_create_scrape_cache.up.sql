-- 006_create_scrape_cache.up.sql
-- Кеш результатов скрейпинга внешних сайтов о дорамах (cache-aside с TTL).
-- Ключ — нормализованные title + host сайта-источника. Данные храним как JSONB
-- (сериализованный scraper.DramaInfo), чтобы не городить отдельную таблицу
-- под каждое поле и не мигрировать схему при каждом новом поле парсера.

CREATE TABLE IF NOT EXISTS scrape_cache (
    id          BIGSERIAL    PRIMARY KEY,
    cache_key   VARCHAR(600) NOT NULL UNIQUE,  -- normalized: "<title>|<site host>"
    title       VARCHAR(500) NOT NULL,
    site_url    TEXT         NOT NULL,
    data        JSONB        NOT NULL,          -- сериализованный DramaInfo
    scraped_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scrape_cache_scraped_at ON scrape_cache (scraped_at);

COMMENT ON TABLE scrape_cache IS 'Cache-aside кеш скрейпинга: отдаём из БД, если запись свежее TTL, иначе идём на сайт и обновляем запись';
COMMENT ON COLUMN scrape_cache.cache_key IS 'lower(title) + "|" + host сайта — уникальный ключ кеша';
