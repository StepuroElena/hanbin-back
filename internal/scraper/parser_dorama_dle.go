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
		queryTitle := extractQueryParam(rawURL, "q")
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
	return strings.Contains(rawURL, "/search")
}

// reDleDoramaLink находит ссылки вида <a href="/12345-slug.html">Название</a>.
// Такие ссылки на странице поиска встречаются несколько раз на карточку
// (обложка + заголовок) — дедуплицируем по href ниже.
var reDleDoramaLink = regexp.MustCompile(`<a[^>]+href="(/\d+-[a-zA-Zа-яА-ЯёЁ0-9-]+\.html)"[^>]*>([^<]{2,150})</a>`)

// bestSearchResult выбирает наиболее подходящую по названию ссылку из результатов поиска.
// Логика идентична doramalandParser.bestSearchResult — доля совпавших токенов
// названия запроса, с порогом 0.5, чтобы не подсовывать нерелевантную дораму.
func (p *dleDoramaParser) bestSearchResult(body, queryTitle string) (string, bool) {
	matches := reDleDoramaLink.FindAllStringSubmatch(body, -1)

	type candidate struct {
		url  string
		name string
	}

	seen := map[string]bool{}
	var candidates []candidate
	for _, m := range matches {
		href := m[1]
		name := strings.TrimSpace(stripTags(m[2]))
		if name == "" || seen[href] || isNavWord(name) {
			continue
		}
		seen[href] = true

		full := href
		if !strings.HasPrefix(full, "http") {
			full = p.baseURL + href
		}
		candidates = append(candidates, candidate{url: full, name: name})
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

	// ── Статус выпуска / перевода ─────────────────────────────────────────────
	// Статус (напр. «Выходит») отображается прямо текстом рядом с названием —
	// общий keyword-хелпер уже это покрывает.
	info.ReleaseTag = parseReleaseTagFromBody(body)
	info.TranslationTag = parseTranslationTagFromBody(body)

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
