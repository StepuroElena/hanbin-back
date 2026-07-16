-- 008_add_poster_url_to_dramas.up.sql
-- Добавляем poster_url — ссылку на постер, извлечённую скрейпером (og:image)
-- при добавлении дорамы. Раньше постер нигде не сохранялся, поэтому карточки
-- и таблица были вынуждены заново искать его только на m.doramatv.one,
-- что не работало для дорам, добавленных с других сайтов.

ALTER TABLE dramas
    ADD COLUMN IF NOT EXISTS poster_url TEXT;

COMMENT ON COLUMN dramas.poster_url IS 'URL постера (og:image), извлечённый скрейпером при добавлении дорамы';
