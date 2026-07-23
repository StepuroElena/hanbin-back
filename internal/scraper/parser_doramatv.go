package scraper

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrNotFound возвращается когда дорама не найдена на сайте.
var ErrNotFound = errors.New("drama not found on this site")

// doramatvParser — парсер для m.doramatv.one и его зеркал.
type doramatvParser struct{}

func (p *doramatvParser) canHandle(host string) bool {
	return strings.Contains(host, "doramatv")
}

func (p *doramatvParser) parse(ctx context.Context, body, rawURL string) (*DramaInfo, error) {
	if p.isSearchPage(body, rawURL) {
		// Извлекаем title из query-параметра q= из rawURL
		queryTitle := extractQueryParam(rawURL, "q")

		var dramaURL string
		var found bool
		if queryTitle != "" {
			// Используем умный поиск с сопоставлением названий
			dramaURL, found = p.bestResultURL(body, queryTitle)
		} else {
			// fallback: первый подходящий результат
			dramaURL, found = p.firstResultURL(body)
		}

		if !found {
			return nil, ErrNotFound
		}
		newBody, _, err := fetch(ctx, dramaURL)
		if err != nil {
			return nil, fmt.Errorf("doramatv: fetch drama page: %w", err)
		}
		info, err := p.parseDramaPage(newBody)
		if err != nil {
			return nil, err
		}
		// Перезаписываем SourceURL на страницу дорамы, а не URL поиска
		info.SourceURL = dramaURL
		return info, nil
	}
	return p.parseDramaPage(body)
}

// extractQueryParam извлекает значение query-параметра из URL.
func extractQueryParam(rawURL, param string) string {
	// Ищем "?q=value" или "&q=value"
	re := regexp.MustCompile(`(?i)[?&]` + regexp.QuoteMeta(param) + `=([^&]+)`)
	m := re.FindStringSubmatch(rawURL)
	if len(m) < 2 {
		return ""
	}
	// URL-декод
	v := m[1]
	v = strings.ReplaceAll(v, "+", " ")
	// Простой декод %XX
	re2 := regexp.MustCompile(`%([0-9A-Fa-f]{2})`)
	v = re2.ReplaceAllStringFunc(v, func(s string) string {
		var b byte
		fmt.Sscanf(s[1:], "%x", &b)
		return string([]byte{b})
	})
	return strings.TrimSpace(v)
}

func (p *doramatvParser) isSearchPage(body, rawURL string) bool {
	return strings.Contains(rawURL, "/search") ||
		strings.Contains(body, "Поиск дорамы")
}

// firstResultURL выбирает из результатов поиска тайл, чьё название
// наиболее близко к запросу. Страница doramatv возвращает нечёткий поиск,
// поэтому первый результат часто нерелевантен.
//
// Структура тайла:
//   <h3><a href="/slug">Русское название</a></h3>
//   <h5>Original Title</h5>  ← внутри .html-popover-holder
//
// Стратегия: собираем все тайлы (.tile) с их slug + ru/en названиями,
// выбираем тот у кого наибольший score совпадения с title запроса.
func (p *doramatvParser) firstResultURL(body string) (string, bool) {
	// fallback: берём первый кандидат с enName из splitTiles
	candidates := splitTileCandidates(body)
	if len(candidates) == 0 {
		return "", false
	}
	// Предпочитаем кандидата с оригинальным названием
	for _, c := range candidates {
		if c.enName != "" {
			return "https://m.doramatv.one" + c.path, true
		}
	}
	return "https://m.doramatv.one" + candidates[0].path, true
}

// bestResultURL — выбирает лучший результат по совпадению с queryTitle.
func (p *doramatvParser) bestResultURL(body, queryTitle string) (string, bool) {
	candidates := splitTileCandidates(body)
	if len(candidates) == 0 {
		return "", false
	}

	qNorm := normTitle(queryTitle)
	qTokens := tokenize(queryTitle)

	best := candidates[0]
	bestRatio := 0.0

	for _, c := range candidates {

		// Точное совпадение (без учёта регистра)
		if normTitle(c.enName) == qNorm || normTitle(c.ruName) == qNorm {
			return "https://m.doramatv.one" + c.path, true
		}

		// Доля совпавших токенов относительно запроса (0..1) — абсолютный счёт с порогом ">0"
		// раньше засчитывал совпадение всего по одному случайному токену и возвращал нерелевантные результаты.
		overlap := tokenOverlap(qTokens, tokenize(c.enName))*3 + tokenOverlap(qTokens, tokenize(c.ruName))
		maxPossible := len(qTokens) * 4 // вес en (x3) + ru (x1) на каждый токен запроса
		ruRatio := 0.0
		if maxPossible > 0 {
			ruRatio = float64(overlap) / float64(maxPossible)
		}

		// Бонус если en-название содержит запрос как подстроку
		if strings.Contains(normTitle(c.enName), qNorm) {
			ruRatio = 1.0
		}

		if ruRatio > bestRatio {
			bestRatio = ruRatio
			best = c
		}
	}

	// Минимальный порог: требуем существенное совпадение, а не любое положительное число.
	if bestRatio < 0.4 {
		return "", false
	}

	return "https://m.doramatv.one" + best.path, true
}

// splitTileCandidates разбивает HTML страницы поиска на тайлы и извлекает
// из каждого: slug, русское название (h3), оригинальное название (h5).
//
// Стратегия разбивки: ищем все вхождения class="tile в HTML и берём кусок
// от текущего до следующего. Это надёжнее чем парсить вложенные теги.
type tileCandidate struct {
	path   string
	ruName string
	enName string
}

func splitTileCandidates(body string) []tileCandidate {
	reH3 := regexp.MustCompile(`<h3[^>]*>\s*<a\s+href="(/[a-z][a-z0-9_-]{1,80})"[^>]*title="([^"]+)"`)
	reH5 := regexp.MustCompile(`<h5[^>]*>([^<]{2,120})</h5>`)

	// Разбиваем body по границам тайлов: ищем позиции class="tile
	markerRe := regexp.MustCompile(`class="tile`)
	positions := markerRe.FindAllStringIndex(body, -1)

	var candidates []tileCandidate
	for i, pos := range positions {
		var chunk string
		if i+1 < len(positions) {
			chunk = body[pos[0]:positions[i+1][0]]
		} else {
			// Последний тайл — берём до конца или следующие 3000 символов
			end := pos[0] + 3000
			if end > len(body) {
				end = len(body)
			}
			chunk = body[pos[0]:end]
		}

		h3m := reH3.FindStringSubmatch(chunk)
		if len(h3m) < 3 {
			continue
		}
		path := h3m[1]

		// Пропускаем служебные пути
		if strings.HasPrefix(path, "/list") || strings.HasPrefix(path, "/internal") ||
			strings.HasPrefix(path, "/search") || strings.HasPrefix(path, "/news") ||
			strings.HasPrefix(path, "/collection") {
			continue
		}

		enName := ""
		if h5m := reH5.FindStringSubmatch(chunk); len(h5m) >= 2 {
			enName = strings.TrimSpace(h5m[1])
		}

		candidates = append(candidates, tileCandidate{
			path:   path,
			ruName: strings.TrimSpace(h3m[2]),
			enName: enName,
		})
	}
	return candidates
}

// normTitle нормализует строку: lowercase + убирает лишние символы.
func normTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	re := regexp.MustCompile(`[^a-zа-яёa-z0-9\s]`)
	s = re.ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// tokenize разбивает строку на слова.
func tokenize(s string) []string {
	s = normTitle(s)
	if s == "" {
		return nil
	}
	tokens := strings.Fields(s)
	// Фильтруем стоп-слова и слишком короткие
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "of": true, "in": true,
		"on": true, "at": true, "to": true, "and": true, "or": true,
		"you": true, "my": true, "is": true, "for": true,
	}
	var result []string
	for _, t := range tokens {
		if len(t) >= 2 && !stopWords[t] {
			result = append(result, t)
		}
	}
	return result
}

// tokenOverlap считает количество общих токенов.
func tokenOverlap(a, b []string) int {
	set := make(map[string]bool, len(a))
	for _, t := range a {
		set[t] = true
	}
	count := 0
	for _, t := range b {
		if set[t] {
			count++
		}
	}
	return count
}

func (p *doramatvParser) parseDramaPage(body string) (*DramaInfo, error) {
	info := &DramaInfo{}

	// ── Заголовок ──────────────────────────────────────────────────────────────
	// og:title надёжнее всего
	if t := metaContent(body, "og:title"); t != "" {
		t = strings.TrimSuffix(strings.TrimSpace(t), " - DoramaTV")
		t = strings.TrimSuffix(t, " — DoramaTV")
		info.Title = strings.TrimSpace(t)
	}
	if info.Title == "" {
		if t := betweenTags(body, "<h1", "</h1>"); t != "" {
			info.Title = stripTags(betweenTags(t, ">", ""))
		}
	}

	// ── Год ────────────────────────────────────────────────────────────────────
	info.ReleaseYear = p.parseYear(body)

	// ── Страна ────────────────────────────────────────────────────────────────
	info.Country = p.parseCountryHero(body)

	// ── Статус выпуска и перевода ─────────────────────────────────────────────
	// Реальная страница показывает эти статусы как простой список лейбл→значение:
	//   Выпуск
	//   Завершён
	//   Перевод
	//   Завершён
	// (подтверждено вживую). Изолируем текст сразу после лейбла и классифицируем
	// именно его — так гораздо надёжнее, чем сканировать ключевые слова по всей
	// странице целиком (там легко словить случайное совпадение).
	info.ReleaseTag = p.parseReleaseTag(body)
	info.TranslationTag = p.parseTranslationTag(body)

	// ── Жанры ─────────────────────────────────────────────────────────────────
	info.Genres = p.parseGenres(body)

	// ── Рейтинг ───────────────────────────────────────────────────────────────
	info.Rating = parseRatingFromBody(body)

	// ── Длительность и серии ──────────────────────────────────────────────────
	// Реальный формат: "15 из 14" (серий) сразу за которым "70 мин." (длительность).
	p.parseEpisodes(body, info)

	// ── Озвучка ───────────────────────────────────────────────────────────────
	// Список студий перевода лежит в отдельном разделе "Переводчики" ниже по
	// странице (не в верхнем инфо-блоке, как считалось раньше) — ссылки вида
	// /list/person/<slug> на каждую студию.
	info.Voiceover = p.parseVoiceover(body)

	// ── Постер ──────────────────────────────────────────────────────────────────
	info.PosterURL = parsePosterURL(body)

	return info, nil
}

// parseYear извлекает год из cr-hero-short-details.
// Приоритет: data-tippy-content="2023 г." → текст спана "2023 - 2024 г." → дата показа
func (p *doramatvParser) parseYear(body string) *int {
	// 1. data-tippy-content="2023 г." или "2023 г.<br>2024 г."
	reTippy := regexp.MustCompile(`data-tippy-content="(\d{4})\s*г\.`)
	if m := reTippy.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseInt(m[1]); ok && v >= 1990 {
			return ptr(v)
		}
	}

	// 2. Текст внутри cr-hero-short-details__item--hoverable: "2023 - 2024 г."
	reHoverable := regexp.MustCompile(`cr-hero-short-details__item--hoverable[^>]*>([^<]{4,30})</`)
	if m := reHoverable.FindStringSubmatch(body); len(m) >= 2 {
		if y := firstMatch(reYear, m[1]); y != "" {
			if v, ok := parseInt(y); ok && v >= 1990 {
				return ptr(v)
			}
		}
	}

	// 3. Ссылка на год в шапке: <a href="/list/year/2026">2026 г.</a> — подтверждено вживую.
	reYearLink := regexp.MustCompile(`(?i)/list/year/(\d{4})"[^>]*>`)
	if m := reYearLink.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseInt(m[1]); ok && v >= 1990 {
			return ptr(v)
		}
	}

	// 4. Дата показа: "с 24.11.2023 по" — берём год из даты начала
	reShowDate := regexp.MustCompile(`с\s+\d{2}\.\d{2}\.(\d{4})\s+по`)
	if m := reShowDate.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseInt(m[1]); ok && v >= 1990 {
			return ptr(v)
		}
	}

	return nil
}

// parseCountryHero извлекает страну из шапки дорамы.
// Подтверждено вживую: <a href="/list/country/south_korea">Южная Корея</a>
func (p *doramatvParser) parseCountryHero(body string) string {
	reCountryLink := regexp.MustCompile(`(?i)/list/country/[^"]+"[^>]*>([^<]{3,30})<`)
	if m := reCountryLink.FindStringSubmatch(body); len(m) >= 2 {
		c := strings.TrimSpace(m[1])
		if c != "" {
			return c
		}
	}
	// Fallback: старый (неподтверждённый) селектор — вдруг где-то встречается
	reCountry := regexp.MustCompile(`cr-hero-short-details__item"[^>]*href="/list/[^"]*">([^<]{3,30})</a>`)
	if m := reCountry.FindStringSubmatch(body); len(m) >= 2 {
		c := strings.TrimSpace(m[1])
		if c != "" {
			return c
		}
	}
	// Fallback: общий парсер
	return parseCountryFromBody(body)
}

// doramatvLabelValue ищет текстовый лейбл (напр. "Выпуск", "Перевод") и возвращает
// текст, идущий сразу за ним — не более пары простых тегов между лейблом и значением.
func doramatvLabelValue(body, label string) string {
	re := regexp.MustCompile(`(?is)` + regexp.QuoteMeta(label) + `\s*(?:</[^>]+>\s*)?(?:<[^>]+>\s*)?([^\n<]{1,60})`)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// classifyStatusValue классифицирует короткое значение статуса ("Завершён", "Выходит" и т.п.)
// на "готово" (released/translated) или "в процессе" (ongoing/translating).
// Возвращает matched=false, если значение не распознано — тогда вызывающий код
// должен откатиться на менее точный фоллбэк.
func classifyStatusValue(v string) (matched bool, ongoing bool) {
	lower := strings.ToLower(v)

	doneWords := []string{"завершён", "завершен", "завершена", "закончен", "закончена", "закончено"}
	for _, w := range doneWords {
		if strings.Contains(lower, w) {
			return true, false
		}
	}

	ongoingWords := []string{
		"выходит", "продолжается", "переводится", "в процессе", "онгоинг",
		"выходят", "не завершён", "не завершен",
	}
	for _, w := range ongoingWords {
		if strings.Contains(lower, w) {
			return true, true
		}
	}

	return false, false
}

// parseReleaseTag определяет статус выпуска по лейблу "Выпуск" на странице.
func (p *doramatvParser) parseReleaseTag(body string) string {
	if v := doramatvLabelValue(body, "Выпуск"); v != "" {
		if matched, ongoing := classifyStatusValue(v); matched {
			if ongoing {
				return "ongoing"
			}
			return "released"
		}
	}
	return parseReleaseTagFromBody(body)
}

// parseTranslationTag определяет статус перевода по лейблу "Перевод" на странице.
func (p *doramatvParser) parseTranslationTag(body string) string {
	if v := doramatvLabelValue(body, "Перевод"); v != "" {
		if matched, ongoing := classifyStatusValue(v); matched {
			if ongoing {
				return "translating"
			}
			return "translated"
		}
	}
	return parseTranslationTagFromBody(body)
}

// parseGenres ищет жанры в ссылках на страницы жанров.
func (p *doramatvParser) parseGenres(body string) []string {
	genreRe := regexp.MustCompile(`(?i)/list/genre[s]?/[^"?#]+["?#][^>]*>([^<]{2,40})</a>`)
	matches := genreRe.FindAllStringSubmatch(body, -1)
	seen := map[string]bool{}
	genres := []string{}
	for _, m := range matches {
		g := strings.TrimSpace(m[1])
		if g != "" && !seen[g] && !strings.EqualFold(g, "все жанры") {
			seen[g] = true
			genres = append(genres, g)
		}
	}
	return genres
}

// parseEpisodes парсит количество серий и длительность одной серии.
// Реальный формат страницы: "15 из 14" (серий вышло из объявленных), сразу
// после которого идёт "70 мин." — оба значения ищем прямо по тексту страницы,
// не полагаясь на конкретный CSS-класс контейнера (тот, что использовался
// раньше, оказался придуманным и не соответствовал реальной разметке).
func (p *doramatvParser) parseEpisodes(body string, info *DramaInfo) {
	reEpTotal := regexp.MustCompile(`(\d+)\s+из\s+(\d+)`)
	loc := reEpTotal.FindStringSubmatchIndex(body)
	if loc != nil {
		if total, ok := parseInt(body[loc[4]:loc[5]]); ok && total > 0 && total < 1000 {
			info.Seasons = []SeasonInfo{{SeasonNumber: 1, EpisodeCount: total}}
		}

		tail := body[loc[1]:]
		if len(tail) > 60 {
			tail = tail[:60]
		}
		if dm := regexp.MustCompile(`(\d+)\s*мин`).FindStringSubmatch(tail); len(dm) >= 2 {
			if v, ok := parseInt(dm[1]); ok && v > 0 && v < 300 {
				info.EpisodeDurationMin = ptr(v)
			}
		}
	}

	if info.EpisodeDurationMin == nil {
		info.EpisodeDurationMin = parseDurationFromBody(body)
	}
}

// parseVoiceover извлекает студии перевода из раздела "Переводчики" — списка
// ссылок вида <a href="/list/person/light_breeze">Light Breeze</a>. Раньше
// парсер искал озвучку в верхнем инфо-блоке рядом с "Серий"/"Канал" — там её
// нет вовсе, реальный список лежит в отдельном разделе значительно ниже.
func (p *doramatvParser) parseVoiceover(body string) string {
	idx := strings.Index(body, "Переводчики")
	if idx == -1 {
		return ""
	}
	window := body[idx+len("Переводчики"):]

	// Раздел заканчивается перед следующим заголовком ("Трейлеры и дополнительные
	// материалы", "Общая оценка" и т.п.) — обрезаем окно там, если нашли.
	if stop := regexp.MustCompile(`(?i)Трейлеры|Общая\s+оценка|Показать\s+ещё`).FindStringIndex(window); stop != nil {
		window = window[:stop[0]]
	}
	if len(window) > 4000 {
		window = window[:4000]
	}

	linkRe := regexp.MustCompile(`(?i)/list/person/[^"']+["'][^>]*>([^<]{2,60})<`)
	matches := linkRe.FindAllStringSubmatch(window, -1)

	seen := map[string]bool{}
	var names []string
	for _, m := range matches {
		name := strings.TrimSpace(m[1])
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}

	joined := strings.Join(names, ", ")
	if len([]rune(joined)) > 255 {
		joined = string([]rune(joined)[:255])
	}
	return joined
}
