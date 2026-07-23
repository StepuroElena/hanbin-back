-- 012_remove_dorama24_streaming_sites.up.sql
-- Dorama24 (dorama24.su) убран из дефолтного списка сайтов (defaultSites в
-- streamingsite.service.go) — сайт на отдельном шаблоне, парсер под него пока не готов.
-- Но удаление из defaultSites влияет только на НОВЫЕ профили (сажается лениво при
-- первом запросе) — профили, уже засеянные до этого изменения, продолжают хранить
-- свою собственную запись Dorama24 в streaming_sites и она никуда не делась.
-- Чистим её задним числом у всех профилей, у кого она есть.
DELETE FROM streaming_sites WHERE url = 'https://dorama24.su';
