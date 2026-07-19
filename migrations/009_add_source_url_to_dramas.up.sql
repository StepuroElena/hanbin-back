-- 009_add_source_url_to_dramas.up.sql
-- Добавляем source_url — точную ссылку на страницу конкретной дорамы у источника.
-- watch_url остаётся ссылкой на сам сайт (из дропдауна, как в модалке добавления),
-- а source_url — опциональное поле, вводится вручную, ведёт на конкретную страницу дорамы.

ALTER TABLE dramas
    ADD COLUMN IF NOT EXISTS source_url TEXT;

COMMENT ON COLUMN dramas.source_url IS 'Точная ссылка на страницу дорамы у источника (опционально, вводится вручную)';
