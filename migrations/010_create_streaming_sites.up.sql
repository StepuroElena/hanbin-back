-- 010_create_streaming_sites.up.sql
-- Сайты для просмотра дорам — раньше жёстко захардкожены во фронте (STREAMING_SITES
-- в AddDramaModal.js), одинаковые для всех пользователей. Теперь каждый профиль имеет
-- свой собственный список в БД: дефолтные 10 сайтов сажаются лениво при первом запросе
-- (см. service.GetAllByProfileID), дальше пользователь может редактировать свой список.

CREATE TABLE IF NOT EXISTS streaming_sites (
    id          BIGSERIAL PRIMARY KEY,
    profile_id  BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    url         TEXT NOT NULL,
    language    TEXT NOT NULL DEFAULT 'ru', -- 'ru' | 'en' | 'multi'
    position    INT NOT NULL DEFAULT 0,     -- порядок отображения в списке
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_streaming_sites_profile_id ON streaming_sites(profile_id);

COMMENT ON TABLE streaming_sites IS 'Персональный список сайтов для просмотра дорам — раньше был захардкожен одинаковым для всех на фронте';
