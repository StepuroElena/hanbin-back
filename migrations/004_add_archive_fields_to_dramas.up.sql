-- 004_add_archive_fields_to_dramas.up.sql
-- Добавляем: архивирование, сезоны, прогресс по сезонам, время серии

ALTER TABLE dramas
    ADD COLUMN IF NOT EXISTS is_archived    BOOLEAN      NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS episode_duration_min SMALLINT CHECK (episode_duration_min IS NULL OR episode_duration_min > 0),
    ADD COLUMN IF NOT EXISTS seasons        JSONB        NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS progress       JSONB        NOT NULL DEFAULT '{"current_episode":0,"seasons":[]}';

COMMENT ON COLUMN dramas.is_archived           IS 'Дорама убрана в архив пользователем';
COMMENT ON COLUMN dramas.episode_duration_min  IS 'Средняя длительность серии в минутах';
COMMENT ON COLUMN dramas.seasons               IS 'Список сезонов: [{season_number, episode_count}]';
COMMENT ON COLUMN dramas.progress              IS 'Прогресс просмотра: {current_episode, seasons:[{season_number, watched_episodes}]}';
