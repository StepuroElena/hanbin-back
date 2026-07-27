-- 020_create_movie_categories.up.sql
-- Персональные категории фильмов — короткие слова/теги, описывающие фильм (напр. «Для вечера»,
-- «Атмосферное», «Экранизация»). Список редактируется на странице настроек и является
-- персональным для каждого профиля — тот же паттерн, что и streaming_sites: дефолтный набор
-- сеется лениво при первом запросе (см. service.GetAllByProfileID).

CREATE TABLE IF NOT EXISTS movie_categories (
    id          BIGSERIAL PRIMARY KEY,
    profile_id  BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    position    INT NOT NULL DEFAULT 0,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_movie_categories_profile_id ON movie_categories(profile_id);

COMMENT ON TABLE movie_categories IS 'Персональный список категорий/тегов фильмов, редактируется в Настройках, аналог streaming_sites';
