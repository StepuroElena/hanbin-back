package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// dleDoramaParser — общий парсер для сайтов на одном и том же (DLE-подобном)
// шаблоне «дорама сайт», который используют несколько доменов из дефолтного
// списка: doramy.club, doramy.info, doram-ru.org.
//
// Общие признаки шаблона (подтверждены реальными страницами всех трёх сайтов):
//   — страницы дорам имеют вид /12345-slug-nazvaniya.html
//   — og:title вида "Название дорама 2024 смотреть онлайн с русской озвучкой"
//   — жанры вынесены на отдельные страницы — либо /genre/<translit-slug>
//     (doramy.club), либо /zhanr/<Кириллица>/ (клоны того же шаблона) —
//     оба варианта уже покрыты общим хелпером parseGenresGeneric()
//     из parser_generic.go
//   — статус выхода отображается словом «Выходит» / «Завершена» —
//     покрыто parseReleaseTagFromBody() из helpers.go
//
// Вместо трёх похожих копипаст-парсеров — один с настраиваемым базовым URL
// для относительных ссылок и хостом для диспетчеризации.
type dleDoramaParser struct {
	// hostMatch — подстрока для поиска в host (регистр не важен, host уже lower-case).
	hostMatch string
	// baseURL — базовый адрес сайта для построения абсолютных ссылок из относительных (без / на конце).
	baseURL string
}

func (p *dleDoramaParser) canHandle(host string) bool {
	return strings.Contains(host, p.hostMatch)
}

func (p *dleDoramaParser) parse(ctx context.Context, body, rawURL string) (*DramaInfo, error) {
	if p.isSearchPage(rawURL) {
		queryTitle := extractQueryParam(rawURL, "s")
		dramaURL, found := p.bestSearchResult(body, queryTitle)
		if !found {
			return nil, ErrNotFound
		}
		newBody, _, err := fetch(ctx, dramaURL)
		if err != nil {
			return nil, fmt.Errorf("%s: fetch drama page: %w", p.hostMatch, err)
		}
		info, err := p.parseDramaPage(newBody)
		if err != nil {
			return nil, err
		}
		// Перезаписываем SourceURL на страницу дорамы, а не поиска
		info.SourceURL = dramaURL
		return info, nil
	}
	return p.parseDramaPage(body)
}

func (p *dleDoramaParser) isSearchPage(rawURL string) bool {
	// Подтверждено вживую (DevTools в браузере на doramy.club): поиск идёт через корневой /?s=,
	// а не /search?q=, как считалось раньше.
	return strings.Contains(rawURL, "?s=") || strings.Contains(rawURL, "&s=")
}

// reDleAnchorOpen находит ОТКРЫВАЮЩИЙ тег ссылки на страницу дорамы вида
// <a href="/12345-slug.html"> или <a href="https://doramy.club/12345-slug.html">.
//
// Подтверждено вживую на doramy.club — реальная карточка результата поиска:
//   <a href="https://doramy.club/57649-yuzhnye-arxivy.html">
//     <img src="..." />
//     <span>Южные архивы</span>
//   </a>
// Раньше regex требовал ОБА условия сразу: (а) относительный href без домена
// и (б) голый текст сразу после ">" — на деле href абсолютный (с доменом), а
// название вложено в <span> после <img>. Ни одно из двух условий не совпадало
// с реальной вёрсткой, поэтому карточки не находились вообще ни разу — отсюда
// «не найдено» независимо от того, что искали. Теперь: домен в href — опционален
// (некапturing группа), а название вытаскиваем из всего содержимого <a>...</a>
// через stripTags, не полагаясь на то, что оно идёт сразу голым текстом.
var reDleAnchorOpen = regexp.MustCompile(`<a[^>]+href="(?:https?://[^/"]+)?(/\d+-[a-zA-Zа-яА-ЯёЁ0-9-]+\.html)"[^>]*>`)

// bestSearchResult выбирает наиболее подходящую по названию ссылку из результатов поиска.
// Логика идентична doramalandParser.bestSearchResult — доля совпавших токенов
// названия запроса, с порогом 0.5, чтобы не подсовывать нерелевантную дораму.
func (p *dleDoramaParser) bestSearchResult(body, queryTitle string) (string, bool) {
	openMatches := reDleAnchorOpen.FindAllStringSubmatchIndex(body, -1)

	type candidate struct {
		url  string
		name string
	}

	seen := map[string]bool{}
	var candidates []candidate
	for _, m := range openMatches {
		tagEnd := m[1]
		hrefStart, hrefEnd := m[2], m[3]
		href := body[hrefStart:hrefEnd]

		// Название может быть обёрнуто в <img>/<span> внутри <a>...</a>, а не идти
		// голым текстом сразу после ">" — берём весь кусок до закрывающего </a> и
		// чистим теги целиком через stripTags, вместо того чтобы угадывать один
		// точный regex-паттерн вложенной разметки.
		closeIdx := strings.Index(body[tagEnd:], "</a>")
		if closeIdx == -1 {
			continue
		}
		inner := body[tagEnd : tagEnd+closeIdx]
		name := strings.TrimSpace(stripTags(inner))

		if name == "" || seen[href] || isNavWord(name) {
			continue
		}
		seen[href] = true
		candidates = append(candidates, candidate{url: p.baseURL + href, name: name})
	}

	if len(candidates) == 0 {
		return "", false
	}
	if queryTitle == "" {
		return candidates[0].url, true
	}

	qNorm := normTitle(queryTitle)
	qTokens := tokenize(queryTitle)

	best := candidates[0]
	bestRatio := 0.0

	for _, c := range candidates {
		if normTitle(c.name) == qNorm {
			return c.url, true
		}

		overlap := tokenOverlap(qTokens, tokenize(c.name))
		ratio := 0.0
		if len(qTokens) > 0 {
			ratio = float64(overlap) / float64(len(qTokens))
		}

		if strings.Contains(normTitle(c.name), qNorm) {
			ratio = 1.0
		}

		if ratio > bestRatio {
			bestRatio = ratio
			best = c
		}
	}

	// Требуем совпадение минимум половины слов запроса — иначе честно «не найдено».
	if bestRatio < 0.5 {
		return "", false
	}
	return best.url, true
}

func (p *dleDoramaParser) parseDramaPage(body string) (*DramaInfo, error) {
	info := &DramaInfo{}

	// ── Заголовок ──────────────────────────────────────────────────────────────
	// og:title обычно имеет вид "Название дорама 2024 смотреть онлайн с русской озвучкой" —
	// обрезаем сервисный хвост.
	if t := metaContent(body, "og:title"); t != "" {
		title := stripTags(t)
		title = regexp.MustCompile(`(?i)\s+дорама\s+\d{4}\s+смотреть\s+онлайн.*$`).ReplaceAllString(title, "")
		title = regexp.MustCompile(`(?i)\s+смотреть\s+онлайн.*$`).ReplaceAllString(title, "")
		info.Title = strings.TrimSpace(title)
	}
	if info.Title == "" {
		if t := betweenTags(body, "<h1", "</h1>"); t != "" {
			info.Title = stripTags(betweenTags(t, ">", ""))
		}
	}

	// ── Год ────────────────────────────────────────────────────────────────────
	// Год чаще всего встречается в og:title ("... 2024 смотреть онлайн") или
	// в подписанном поле "Год:" в сайдбаре с деталями.
	yearCandidates := []string{
		betweenTags(body, "Год:", "<"),
		betweenTags(body, "Год выхода:", "<"),
		metaContent(body, "og:title"),
	}
	for _, c := range yearCandidates {
		if y := firstMatch(reYear, c); y != "" {
			if v, ok := parseInt(y); ok && v >= 1990 {
				info.ReleaseYear = ptr(v)
				break
			}
		}
	}
	// Фоллбэк: на doramy.club год идёт простым текстом рядом с названием (напр. "Китай, 2026"),
	// без отдельного лейбла "Год:" — берём первый год из начала страницы (там, где обычно
	// лежат детали дорамы, а не случайный год из подвала/футера).
	if info.ReleaseYear == nil {
		window := body
		if len(window) > 3000 {
			window = window[:3000]
		}
		if y := firstMatch(reYear, window); y != "" {
			if v, ok := parseInt(y); ok && v >= 1990 {
				info.ReleaseYear = ptr(v)
			}
		}
	}

	// ── Статус выпуска / перевода ─────────────────────────────────────────────
	// Статус выпуска (напр. «Выходит») отображается прямо текстом рядом с названием —
	// общий keyword-хелпер это неплохо покрывает.
	//
	// Для статуса ПЕРЕВОДА независимого надёжного сигнала на этих сайтах нет —
	// раньше здесь тоже стоял общий keyword-скан по всему телу страницы, но он
	// слишком легко ловит случайные совпадения (см. разбор той же проблемы в
	// doramalandParser). Вместо этого выводим статус перевода из уже определённого
	// статуса выпуска: завершённая дорама чаще всего уже переведена полностью,
	// а идущая — переводится вместе с выходом новых серий.
	info.ReleaseTag = parseReleaseTagFromBody(body)
	if info.ReleaseTag == "released" {
		info.TranslationTag = "translated"
	} else {
		info.TranslationTag = "translating"
	}

	// ── Озвучка ────────────────────────────────────────────────────────────────
	// "Озвучка: X, Y, Z" — тот же формат лейбла, что подтверждён вживую на dorama.land;
	// если на конкретном сайте из этого семейства такого лейбла нет — просто остаётся пусто.
	info.Voiceover = parseVoiceoverLabelled(body)

	// ── Жанры ─────────────────────────────────────────────────────────────────
	// Ссылки вида /genre/melodrama или /zhanr/Комедия/ — оба варианта уже покрыты
	// общим хелпером parseGenresGeneric.
	info.Genres = parseGenresGeneric(body)

	// ── Страна ────────────────────────────────────────────────────────────────
	reCountryLabelLink := regexp.MustCompile(`(?i)Страна[^:<]*:\s*</[^>]+>\s*<a[^>]*>([^<]+)<`)
	reCountryLabelText := regexp.MustCompile(`(?i)Страна[^:<]*:\s*([^\n<,]{2,40})`)
	if m := reCountryLabelLink.FindStringSubmatch(body); len(m) >= 2 {
		info.Country = strings.TrimSpace(m[1])
	} else if m := reCountryLabelText.FindStringSubmatch(body); len(m) >= 2 {
		info.Country = strings.TrimSpace(m[1])
	}
	if info.Country == "" {
		info.Country = parseCountryFromBody(body)
	}

	// ── Рейтинг ───────────────────────────────────────────────────────────────
	info.Rating = parseRatingFromBody(body)

	// ── Длительность ──────────────────────────────────────────────────────────
	info.EpisodeDurationMin = parseDurationFromBody(body)

	// ── Сезоны / серии ────────────────────────────────────────────────────────
	info.Seasons = parseSeasonsGeneric(body)

	// ── Постер ──────────────────────────────────────────────────────────────────
	info.PosterURL = parsePosterURL(body)

	return info, nil
}
