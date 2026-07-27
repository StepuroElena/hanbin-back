-- 014_add_watch_status_to_movies.up.sql
-- Статус просмотра фильма — минимальная версия из двух значений (в отличие от дорам,
-- где их четыре): "запланирован" или "просмотрен". Нужно для счётчиков на странице фильмов.

ALTER TABLE movies
    ADD COLUMN IF NOT EXISTS watch_status VARCHAR(20) NOT NULL DEFAULT 'planned'
        CHECK (watch_status IN ('planned', 'watched'));
