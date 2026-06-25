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
	bestScore := -1

	for _, c := range candidates {
		score := 0

		// Точное совпадение (без учёта регистра)
		if normTitle(c.enName) == qNorm || normTitle(c.ruName) == qNorm {
			return "https://m.doramatv.one" + c.path, true
		}

		// Считаем общие токены с en-названием (приоритет) и ru-названием
		score += tokenOverlap(qTokens, tokenize(c.enName)) * 3
		score += tokenOverlap(qTokens, tokenize(c.ruName))

		// Бонус если en-название содержит запрос как подстроку
		if strings.Contains(normTitle(c.enName), qNorm) {
			score += 10
		}

		if score > bestScore {
			bestScore = score
			best = c
		}
	}

	// Минимальный порог: хотя бы одно общее слово
	if bestScore <= 0 {
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
	// Структура: <span class="cr-hero-short-details__item cr-hero-short-details__item--hoverable"
	//              data-tippy-content="2023 г.<br>2024 г." ...>2023 - 2024 г.</span>
	// Берём год из data-tippy-content или из текста спана
	info.ReleaseYear = p.parseYear(body)

	// ── Страна ────────────────────────────────────────────────────────────────
	// <a class="cr-hero-short-details__item" href="/list/...">Южная Корея</a>
	info.Country = p.parseCountryHero(body)

	// ── Статус выпуска ─────────────────────────────────────────────────────────
	// <div class="cr-info-details-item__title">Выпуск</div>
	// <span class="cr-info-details-item__status" data-production-status="FINISHED">Завершён</span>
	info.ReleaseTag = p.parseReleaseTag(body)

	// ── Статус перевода ────────────────────────────────────────────────────────
	// <div class="cr-info-details-item__title">Перевод</div>
	// <span ... data-translation-status="FINISHED">Завершён</span>
	info.TranslationTag = p.parseTranslationTag(body)

	// ── Жанры ─────────────────────────────────────────────────────────────────
	info.Genres = p.parseGenres(body)

	// ── Рейтинг ───────────────────────────────────────────────────────────────
	info.Rating = parseRatingFromBody(body)

	// ── Длительность и серии ──────────────────────────────────────────────────
	// "16 из 16 65 мин." из блока с title "Серий"
	p.parseEpisodes(body, info)

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

	// 3. Дата показа: "с 24.11.2023 по" — берём год из даты начала
	reShowDate := regexp.MustCompile(`с\s+\d{2}\.\d{2}\.(\d{4})\s+по`)
	if m := reShowDate.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseInt(m[1]); ok && v >= 1990 {
			return ptr(v)
		}
	}

	return nil
}

// parseCountryHero извлекает страну из шапки дорамы.
// <a class="cr-hero-short-details__item" ...>Южная Корея</a>
func (p *doramatvParser) parseCountryHero(body string) string {
	// Ищем ссылки в блоке cr-hero-short-details
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

// parseReleaseTag читает data-production-status из блока деталей.
// FINISHED → "released", AIRING / IN_PROGRESS → "ongoing", ANNOUNCED → "planned"
func (p *doramatvParser) parseReleaseTag(body string) string {
	reStatus := regexp.MustCompile(`data-production-status="([^"]+)"`)
	if m := reStatus.FindStringSubmatch(body); len(m) >= 2 {
		switch strings.ToUpper(m[1]) {
		case "FINISHED":
			return "released"
		case "AIRING", "IN_PROGRESS", "ONGOING":
			return "ongoing"
		case "ANNOUNCED", "PLANNED":
			return "planned"
		}
	}
	// Fallback: текстовый поиск
	return parseReleaseTagFromBody(body)
}

// parseTranslationTag читает data-translation-status.
func (p *doramatvParser) parseTranslationTag(body string) string {
	reStatus := regexp.MustCompile(`data-translation-status="([^"]+)"`)
	if m := reStatus.FindStringSubmatch(body); len(m) >= 2 {
		switch strings.ToUpper(m[1]) {
		case "FINISHED":
			return "translated"
		case "IN_PROGRESS", "ONGOING", "AIRING":
			return "translating"
		}
	}
	return parseTranslationTagFromBody(body)
}

// parseGenres ищет жанры в ссылках на страницы жанров.
func (p *doramatvParser) parseGenres(body string) []string {
	genreRe := regexp.MustCompile(`(?i)/list/genres/[^"?#]+["?#][^>]*>([^<]{2,40})</a>`)
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

// parseEpisodes парсит количество серий и длительность из блока "Серий".
// Формат: "16 из 16 65 мин."
func (p *doramatvParser) parseEpisodes(body string, info *DramaInfo) {
	// Ищем блок с заголовком "Серий"
	reSeriesBlock := regexp.MustCompile(`(?i)cr-info-details-item__title">Серий</div>\s*<div[^>]*>([\s\S]{0,300}?)</div>\s*</div>`)
	m := reSeriesBlock.FindStringSubmatch(body)
	if len(m) < 2 {
		return
	}
	block := stripTags(m[1])

	// "16 из 16" — берём второе число как total
	reEpTotal := regexp.MustCompile(`(\d+)\s+из\s+(\d+)`)
	if em := reEpTotal.FindStringSubmatch(block); len(em) >= 3 {
		if total, ok := parseInt(em[2]); ok && total > 0 {
			info.Seasons = []SeasonInfo{{SeasonNumber: 1, EpisodeCount: total}}
		}
	} else {
		// Просто число серий без "из"
		if ep := firstMatch(regexp.MustCompile(`(\d+)`), block); ep != "" {
			if v, ok := parseInt(ep); ok && v > 0 && v < 1000 {
				info.Seasons = []SeasonInfo{{SeasonNumber: 1, EpisodeCount: v}}
			}
		}
	}

	// Длительность: "65 мин."
	reDur := regexp.MustCompile(`(\d+)\s*мин`)
	if dm := reDur.FindStringSubmatch(block); len(dm) >= 2 {
		if v, ok := parseInt(dm[1]); ok && v > 0 && v < 300 {
			info.EpisodeDurationMin = ptr(v)
		}
	}
}
