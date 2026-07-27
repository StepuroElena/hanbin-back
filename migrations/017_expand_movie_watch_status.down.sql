-- 017_expand_movie_watch_status.down.sql

UPDATE movies SET watch_status = 'watched' WHERE watch_status = 'completed';
UPDATE movies SET watch_status = 'planned' WHERE watch_status IN ('watching', 'dropped');

ALTER TABLE movies DROP CONSTRAINT IF EXISTS movies_watch_status_check;
ALTER TABLE movies ADD CONSTRAINT movies_watch_status_check
    CHECK (watch_status IN ('planned', 'watched'));
