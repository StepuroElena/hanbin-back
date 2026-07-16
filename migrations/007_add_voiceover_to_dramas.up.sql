-- 007_add_voiceover_to_dramas.up.sql
-- Добавляем поле "Озвучка" — студия/автор озвучки, парсится с сайта-источника
-- вместе с остальными данными о дораме (жанр, год, серии и т.д.)

ALTER TABLE dramas
    ADD COLUMN IF NOT EXISTS voiceover VARCHAR(255);

COMMENT ON COLUMN dramas.voiceover IS 'Озвучка/студия перевода, парсится с сайта-источника (например: LostFilm, Дубляж, оригинал+субтитры)';
