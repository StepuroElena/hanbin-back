-- scripts/seed_n63223304.sql
--
-- Наполняет библиотеку пользователя n63223304@gmail.com тестовыми данными:
-- 15 дорам + 20 фильмов, с разнообразными статусами/жанрами/странами —
-- чтобы на дашборде (статистика, фильтры, Random Picker) было что показывать.
--
-- Запуск:
--   psql -h localhost -U elenastepuro -d hanbin -f scripts/seed_n63223304.sql
--
-- Требование: пользователь с этим email должен уже существовать в profiles
-- (зарегистрирован через UI/API) — скрипт не создаёт аккаунт, только данные.

DO $$
DECLARE
  v_profile_id BIGINT;
BEGIN
  SELECT id INTO v_profile_id FROM profiles WHERE email = 'n63223304@gmail.com';
  IF v_profile_id IS NULL THEN
    RAISE EXCEPTION 'Профиль с email n63223304@gmail.com не найден. Сначала зарегистрируй пользователя через UI (форма регистрации) или API, потом запусти скрипт снова.';
  END IF;
  RAISE NOTICE 'Найден profile_id = %, наполняю библиотеку...', v_profile_id;
END $$;

-- ─────────────────────────────────────────────
-- ДОРАМЫ (15 шт.)
-- ─────────────────────────────────────────────

INSERT INTO dramas (
  profile_id, title, watch_url, source_url, release_year,
  release_tag, translation_tag, genre, rating,
  watch_status, country,
  is_archived, episode_duration_min, voiceover, poster_url, seasons, progress,
  created_at, updated_at
)
SELECT p.id, v.title, v.watch_url, v.source_url, v.release_year,
       v.release_tag::drama_release_tag, v.translation_tag::drama_translation_tag, v.genre, v.rating,
       v.watch_status::drama_watch_status, v.country,
       FALSE, v.episode_duration_min, v.voiceover, v.poster_url, v.seasons::jsonb, v.progress::jsonb,
       NOW() - v.created_offset, NOW() - v.updated_offset
FROM (SELECT id FROM profiles WHERE email = 'n63223304@gmail.com') p
CROSS JOIN (VALUES
  -- title, watch_url, source_url, release_year, release_tag, translation_tag, genre, rating, watch_status, country,
  -- episode_duration_min, voiceover, poster_url, seasons, progress, created_offset, updated_offset
  ('My Demon', 'https://example.com/my-demon', '', 2023, 'released', 'translated', 'Romance', 8.4, 'watching', 'kr',
    70, 'Дубляж', 'https://images.unsplash.com/photo-1535016120720-40c646be5580?w=400&q=80',
    '[{"season_number":1,"episode_count":16}]', '{"current_episode":10,"seasons":[{"season_number":1,"watched_episodes":10}]}',
    INTERVAL '12 days', INTERVAL '1 hours'),
  ('The Story of Hua Zhi', 'https://example.com/hua-zhi', '', 2024, 'released', 'translating', 'Historical', NULL, 'watching', 'cn',
    45, '', 'https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=400&q=80',
    '[{"season_number":1,"episode_count":40}]', '{"current_episode":12,"seasons":[{"season_number":1,"watched_episodes":12}]}',
    INTERVAL '8 days', INTERVAL '6 hours'),
  ('Lovely Runner', 'https://example.com/lovely-runner', '', 2024, 'released', 'translated', 'Romance', 9.6, 'watching', 'kr',
    70, 'Субтитры', 'https://images.unsplash.com/photo-1490730141103-6cac27aaab94?w=400&q=80',
    '[{"season_number":1,"episode_count":16}]', '{"current_episode":14,"seasons":[{"season_number":1,"watched_episodes":14}]}',
    INTERVAL '5 days', INTERVAL '30 minutes'),
  ('Marry My Husband', 'https://example.com/marry-my-husband', '', 2024, 'released', 'translated', 'Thriller', 8.8, 'watching', 'kr',
    65, 'Дубляж', 'https://images.unsplash.com/photo-1476514525535-07fb3b4ae5f1?w=400&q=80',
    '[{"season_number":1,"episode_count":16}]', '{"current_episode":7,"seasons":[{"season_number":1,"watched_episodes":7}]}',
    INTERVAL '20 days', INTERVAL '2 days'),
  ('Vincenzo', 'https://example.com/vincenzo', '', 2021, 'released', 'translating', 'Action', 9.0, 'watching', 'kr',
    80, '', 'https://images.unsplash.com/photo-1547036967-23d11aacaee0?w=400&q=80',
    '[{"season_number":1,"episode_count":20}]', '{"current_episode":15,"seasons":[{"season_number":1,"watched_episodes":15}]}',
    INTERVAL '3 days', INTERVAL '4 hours'),
  ('Queen of Tears', 'https://example.com/queen-of-tears', '', 2024, 'released', 'translated', 'Romance', 9.4, 'completed', 'kr',
    75, 'Дубляж', 'https://images.unsplash.com/photo-1557804506-669a67965ba0?w=400&q=80',
    '[{"season_number":1,"episode_count":16}]', '{"current_episode":16,"seasons":[{"season_number":1,"watched_episodes":16}]}',
    INTERVAL '30 days', INTERVAL '2 hours'),
  ('Crash Landing on You', 'https://example.com/cloy', '', 2019, 'released', 'translated', 'Romance', 9.8, 'completed', 'kr',
    80, 'Субтитры', 'https://images.unsplash.com/photo-1551269901-5c5e14c25df7?w=400&q=80',
    '[{"season_number":1,"episode_count":16}]', '{"current_episode":16,"seasons":[{"season_number":1,"watched_episodes":16}]}',
    INTERVAL '90 days', INTERVAL '26 hours'),
  ('Nirvana in Fire', 'https://example.com/nirvana-in-fire', '', 2015, 'released', 'translated', 'Historical', 9.6, 'completed', 'cn',
    45, '', 'https://images.unsplash.com/photo-1526374965328-7f61d4dc18c5?w=400&q=80',
    '[{"season_number":1,"episode_count":54}]', '{"current_episode":54,"seasons":[{"season_number":1,"watched_episodes":54}]}',
    INTERVAL '120 days', INTERVAL '96 hours'),
  ('Alchemy of Souls', 'https://example.com/alchemy-of-souls', '', 2022, 'released', 'translated', 'Fantasy', 8.6, 'completed', 'kr',
    75, 'Дубляж', 'https://images.unsplash.com/photo-1519895709498-ce3c5fa1a100?w=400&q=80',
    '[{"season_number":2,"episode_count":20}]', '{"current_episode":20,"seasons":[{"season_number":1,"watched_episodes":20},{"season_number":2,"watched_episodes":20}]}',
    INTERVAL '60 days', INTERVAL '48 hours'),
  ('Reply 1988', 'https://example.com/reply-1988', '', 2015, 'released', 'translated', 'Comedy', 9.8, 'completed', 'kr',
    90, 'Субтитры', 'https://images.unsplash.com/photo-1535016120720-40c646be5580?w=400&q=80',
    '[{"season_number":1,"episode_count":20}]', '{"current_episode":20,"seasons":[{"season_number":1,"watched_episodes":20}]}',
    INTERVAL '150 days', INTERVAL '120 hours'),
  ('Twenty-Five Twenty-One', 'https://example.com/2521', '', 2022, 'released', 'translating', 'Romance', NULL, 'planned', 'kr',
    60, '', 'https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=400&q=80',
    '[{"season_number":1,"episode_count":16}]', '{"current_episode":0,"seasons":[]}',
    INTERVAL '3 days', INTERVAL '3 days'),
  ('The Untamed', 'https://example.com/the-untamed', '', 2019, 'released', 'translating', 'Fantasy', NULL, 'planned', 'cn',
    45, '', 'https://images.unsplash.com/photo-1490730141103-6cac27aaab94?w=400&q=80',
    '[{"season_number":1,"episode_count":50}]', '{"current_episode":0,"seasons":[]}',
    INTERVAL '2 days', INTERVAL '2 days'),
  ('Midnight Diner', 'https://example.com/midnight-diner', '', 2016, 'released', 'translating', 'Drama', NULL, 'planned', 'other',
    25, '', 'https://images.unsplash.com/photo-1476514525535-07fb3b4ae5f1?w=400&q=80',
    '[{"season_number":1,"episode_count":10}]', '{"current_episode":0,"seasons":[]}',
    INTERVAL '1 days', INTERVAL '1 days'),
  ('Sky Castle', 'https://example.com/sky-castle', '', 2018, 'released', 'translated', 'Mystery', 6.0, 'dropped', 'kr',
    80, 'Дубляж', 'https://images.unsplash.com/photo-1547036967-23d11aacaee0?w=400&q=80',
    '[{"season_number":1,"episode_count":20}]', '{"current_episode":6,"seasons":[{"season_number":1,"watched_episodes":6}]}',
    INTERVAL '45 days', INTERVAL '40 days'),
  ('Word of Honor', 'https://example.com/word-of-honor', '', 2021, 'released', 'translated', 'Historical', 5.5, 'dropped', 'cn',
    45, '', 'https://images.unsplash.com/photo-1557804506-669a67965ba0?w=400&q=80',
    '[{"season_number":1,"episode_count":36}]', '{"current_episode":8,"seasons":[{"season_number":1,"watched_episodes":8}]}',
    INTERVAL '70 days', INTERVAL '65 days')
) AS v(title, watch_url, source_url, release_year, release_tag, translation_tag, genre, rating, watch_status, country,
       episode_duration_min, voiceover, poster_url, seasons, progress, created_offset, updated_offset);

-- ─────────────────────────────────────────────
-- ФИЛЬМЫ (20 шт.)
-- ─────────────────────────────────────────────

INSERT INTO movies (
  profile_id, title, genre, country, category, release_year, watch_status, is_archived,
  created_at, updated_at
)
SELECT p.id, v.title, v.genre, v.country, '', v.release_year, v.watch_status::VARCHAR(20), FALSE,
       NOW() - v.created_offset, NOW() - v.updated_offset
FROM (SELECT id FROM profiles WHERE email = 'n63223304@gmail.com') p
CROSS JOIN (VALUES
  ('Parasite', 'Drama', 'kr', 2019, 'completed', INTERVAL '100 days', INTERVAL '90 days'),
  ('Decision to Leave', 'Mystery', 'kr', 2022, 'completed', INTERVAL '80 days', INTERVAL '70 days'),
  ('The Handmaiden', 'Thriller', 'kr', 2016, 'completed', INTERVAL '95 days', INTERVAL '85 days'),
  ('Oldboy', 'Thriller', 'kr', 2003, 'completed', INTERVAL '200 days', INTERVAL '190 days'),
  ('Burning', 'Mystery', 'kr', 2018, 'completed', INTERVAL '60 days', INTERVAL '55 days'),
  ('Train to Busan', 'Horror', 'kr', 2016, 'completed', INTERVAL '150 days', INTERVAL '140 days'),
  ('Extreme Job', 'Comedy', 'kr', 2019, 'completed', INTERVAL '40 days', INTERVAL '35 days'),
  ('Spirited Away', 'Fantasy', 'jp', 2001, 'completed', INTERVAL '210 days', INTERVAL '200 days'),
  ('The Roundup', 'Action', 'kr', 2022, 'watching', INTERVAL '5 days', INTERVAL '1 days'),
  ('Hunt', 'Action', 'kr', 2022, 'watching', INTERVAL '4 days', INTERVAL '2 days'),
  ('Along with the Gods', 'Fantasy', 'kr', 2017, 'watching', INTERVAL '6 days', INTERVAL '3 days'),
  ('Broker', 'Drama', 'kr', 2022, 'watching', INTERVAL '7 days', INTERVAL '1 days'),
  ('In the Mood for Love', 'Romance', 'cn', 2000, 'planned', INTERVAL '2 days', INTERVAL '2 days'),
  ('Crouching Tiger Hidden Dragon', 'Action', 'cn', 2000, 'planned', INTERVAL '3 days', INTERVAL '3 days'),
  ('Farewell My Concubine', 'Drama', 'cn', 1993, 'planned', INTERVAL '1 days', INTERVAL '1 days'),
  ('Shoplifters', 'Drama', 'jp', 2018, 'planned', INTERVAL '2 days', INTERVAL '2 days'),
  ('Drive My Car', 'Drama', 'jp', 2021, 'planned', INTERVAL '1 days', INTERVAL '1 days'),
  ('Your Name', 'Fantasy', 'jp', 2016, 'planned', INTERVAL '4 days', INTERVAL '4 days'),
  ('I Saw the Devil', 'Thriller', 'kr', 2010, 'dropped', INTERVAL '50 days', INTERVAL '45 days'),
  ('A Tale of Two Sisters', 'Horror', 'kr', 2003, 'dropped', INTERVAL '55 days', INTERVAL '50 days')
) AS v(title, genre, country, release_year, watch_status, created_offset, updated_offset);

-- ─────────────────────────────────────────────
-- ИТОГ
-- ─────────────────────────────────────────────

DO $$
DECLARE
  v_profile_id BIGINT;
  v_dramas_count INT;
  v_movies_count INT;
BEGIN
  SELECT id INTO v_profile_id FROM profiles WHERE email = 'n63223304@gmail.com';
  SELECT COUNT(*) INTO v_dramas_count FROM dramas WHERE profile_id = v_profile_id;
  SELECT COUNT(*) INTO v_movies_count FROM movies WHERE profile_id = v_profile_id;
  RAISE NOTICE 'Готово. У профиля % теперь % дорам и % фильмов.', v_profile_id, v_dramas_count, v_movies_count;
END $$;
