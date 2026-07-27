-- 017_expand_movie_watch_status.up.sql
-- Расширяем watch_status до четырёх значений, как у дорам: planned, watching, completed, dropped
-- (нужно для фильтр-чипсов «Все/Смотрю/Просмотрено/Запланировано/Брошено» на странице фильмов).
-- Старое значение 'watched' переименовываем в 'completed' для единообразия с терминологией дорам.

UPDATE movies SET watch_status = 'completed' WHERE watch_status = 'watched';

ALTER TABLE movies DROP CONSTRAINT IF EXISTS movies_watch_status_check;
ALTER TABLE movies ADD CONSTRAINT movies_watch_status_check
    CHECK (watch_status IN ('planned', 'watching', 'completed', 'dropped'));
