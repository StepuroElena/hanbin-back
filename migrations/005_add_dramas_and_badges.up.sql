-- 005_add_dramas_and_badges.up.sql
-- Добавляем таблицы для дорам пользователя и бэйджей.

-- 1. Дорамы
CREATE TABLE IF NOT EXISTS dramas (
    id              BIGSERIAL    PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(500) NOT NULL,
    year            SMALLINT,
    genre           VARCHAR(255),
    country         VARCHAR(100),
    doramatv_rating NUMERIC(3,1),        -- рейтинг с doramatv.one, например 8.7
    watch_status    VARCHAR(50)  NOT NULL DEFAULT 'plan',  -- watching | completed | plan | dropped
    current_episode INT          NOT NULL DEFAULT 0,
    total_episodes  INT,
    doramatv_url    VARCHAR(1000),       -- ссылка на страницу на doramatv.one
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dramas_user_id ON dramas (user_id);

-- 2. Теги дорамы (выходит/выпущен, переведён/переводится)
CREATE TABLE IF NOT EXISTS drama_tags (
    id       BIGSERIAL   PRIMARY KEY,
    drama_id BIGINT      NOT NULL REFERENCES dramas(id) ON DELETE CASCADE,
    tag      VARCHAR(100) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_drama_tags_drama_id ON drama_tags (drama_id);

-- 3. Бэйджи
CREATE TABLE IF NOT EXISTS badges (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code        VARCHAR(100) NOT NULL,   -- уникальный код: drama_queen, k_drama_fan, ...
    name        VARCHAR(255) NOT NULL,
    description VARCHAR(500),
    icon        VARCHAR(50),             -- emoji или имя иконки
    earned_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT badges_user_code_unique UNIQUE (user_id, code)
);

CREATE INDEX IF NOT EXISTS idx_badges_user_id ON badges (user_id);
