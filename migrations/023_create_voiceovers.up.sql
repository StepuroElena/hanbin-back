-- 023_create_voiceovers.up.sql
-- Озвучки (студии перевода) — раньше были захардкожены во фронте (VOICEOVER_OPTIONS в
-- utils/helpers.js), одинаковые для всех. Теперь список живёт в БД и редактируется в Настройках.
--
-- Модель:
--   • voiceovers.profile_id IS NULL      — дефолтная озвучка, общая для всех пользователей.
--     Её нельзя переименовать или удалить — только выключить/включить у себя тоглом.
--     Новую дефолтную озвучку для всех добавляем одной строкой INSERT — без релиза кода.
--   • voiceovers.profile_id = <профиль>   — собственная озвучка пользователя:
--     её можно переименовать, удалить, выключить.
--   • voiceover_settings                 — персональный тогл вкл/выкл (profile × voiceover).
--     Нет строки = включена. Так дефолты не копируются каждому пользователю,
--     а храним только отличия от умолчания.

CREATE TABLE IF NOT EXISTS voiceovers (
    id          BIGSERIAL PRIMARY KEY,
    profile_id  BIGINT REFERENCES profiles(id) ON DELETE CASCADE, -- NULL = дефолтная для всех
    name        VARCHAR(255) NOT NULL,
    position    INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_voiceovers_profile_id ON voiceovers(profile_id);

-- Уникальность названий без учёта регистра: среди дефолтных и внутри списка каждого профиля.
-- (Пересечение своей озвучки с дефолтной проверяет сервис.)
CREATE UNIQUE INDEX IF NOT EXISTS uq_voiceovers_default_name
    ON voiceovers (lower(name)) WHERE profile_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_voiceovers_profile_name
    ON voiceovers (profile_id, lower(name)) WHERE profile_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS voiceover_settings (
    profile_id    BIGINT NOT NULL REFERENCES profiles(id)   ON DELETE CASCADE,
    voiceover_id  BIGINT NOT NULL REFERENCES voiceovers(id) ON DELETE CASCADE,
    enabled       BOOLEAN NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (profile_id, voiceover_id)
);

-- Дефолтный набор — тот же, что был в VOICEOVER_OPTIONS на фронте.
INSERT INTO voiceovers (profile_id, name, position) VALUES
    (NULL, 'Light Breeze', 0),
    (NULL, 'SlothSound',   1),
    (NULL, 'DubLik.TV',    2),
    (NULL, 'SoftBox',      3),
    (NULL, 'AniMaunt',     4),
    (NULL, 'STEPonee',     5)
ON CONFLICT DO NOTHING;

COMMENT ON TABLE voiceovers IS 'Озвучки: profile_id NULL — дефолтные для всех (только тогл), иначе — собственные озвучки профиля';
COMMENT ON TABLE voiceover_settings IS 'Персональный тогл вкл/выкл озвучки; нет строки = включена';
