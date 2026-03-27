package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// doramalandParser — парсер для dorama.land
type doramalandParser struct{}

func (p *doramalandParser) canHandle(host string) bool {
	return strings.Contains(host, "dorama.land")
}

func (p *doramalandParser) parse(ctx context.Context, body, rawURL string) (*DramaInfo, error) {
	if p.isSearchPage(rawURL) {
		queryTitle := extractQueryParam(rawURL, "q")
		dramaURL, found := p.bestSearchResult(body, queryTitle)
		if !found {
			return nil, ErrNotFound
		}
		newBody, _, err := fetch(ctx, dramaURL)
		if err != nil {
			return nil, fmt.Errorf("doramaland: fetch drama page: %w", err)
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

func (p *doramalandParser) isSearchPage(rawURL string) bool {
	return strings.Contains(rawURL, "/search")
}

// bestSearchResult — выбирает лучший результат из .search-item блоков.
// Реальная структура dorama.land:
//   <div class="search-item">
//     <a href="/slug" class="search-item-wrap">
//       <h2 class="search-item__title h3">Русское название</h2>
//       <span class="search-item__alternative-title"> EN / Alt</span>
//     </a>
//   </div>
func (p *doramalandParser) bestSearchResult(body, queryTitle string) (string, bool) {
	// Разбиваем body по позициям маркера class="search-item"
	// Тот же подход, что в splitTileCandidates для doramatv
	markerRe := regexp.MustCompile(`class="search-item"`)
	positions := markerRe.FindAllStringIndex(body, -1)

	// Из каждого куска извлекаем href (возможные порядки атрибутов), ru, en
	// href может быть до или после class в теге <a>:
	//   <a href="/slug" class="search-item-wrap">   (href первый)
	//   <a class="search-item-wrap" href="/slug">   (class первый)
	reHref := regexp.MustCompile(`<a[^>]*href="(/[^"]{2,100})"[^>]*class="search-item-wrap"|<a[^>]*class="search-item-wrap"[^>]*href="(/[^"]{2,100})"`)
	reTitle := regexp.MustCompile(`class="search-item__title[^"]*"[^>]*>([^<]+)<`)
	reAlt := regexp.MustCompile(`class="search-item__alternative-title"[^>]*>\s*([^<]+)<`)

	type candidate struct {
		url    string
		ruName string
		enName string
	}

	var candidates []candidate
	for i, pos := range positions {
		var chunk string
		if i+1 < len(positions) {
			chunk = body[pos[0]:positions[i+1][0]]
		} else {
			end := pos[0] + 2000
			if end > len(body) {
				end = len(body)
			}
			chunk = body[pos[0]:end]
		}

		hm := reHref.FindStringSubmatch(chunk)
		if len(hm) < 2 {
			continue
		}
		href := hm[1]
		if href == "" && len(hm) > 2 {
			href = hm[2]
		}
		href = strings.TrimSpace(href)
		if href == "" {
			continue
		}

		ru := ""
		if tm := reTitle.FindStringSubmatch(chunk); len(tm) >= 2 {
			ru = strings.TrimSpace(tm[1])
		}
		en := ""
		if am := reAlt.FindStringSubmatch(chunk); len(am) >= 2 {
			en = strings.TrimSpace(am[1])
		}

		// Строим полный URL
		url := href
		if !strings.HasPrefix(url, "http") {
			url = "https://dorama.land" + href
		}
		candidates = append(candidates, candidate{url: url, ruName: ru, enName: en})
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
	bestScore := -1

	for _, c := range candidates {
		// Точное совпадение
		if normTitle(c.enName) == qNorm || normTitle(c.ruName) == qNorm {
			return c.url, true
		}

		score := 0
		// Совпадение токенов: EN-альтернативные названия через "/"
		for _, enPart := range strings.Split(c.enName, "/") {
			score += tokenOverlap(qTokens, tokenize(enPart)) * 3
		}
		score += tokenOverlap(qTokens, tokenize(c.ruName))

		if strings.Contains(normTitle(c.enName), qNorm) {
			score += 10
		}

		if score > bestScore {
			bestScore = score
			best = c
		}
	}

	if bestScore <= 0 {
		return "", false
	}
	return best.url, true
}

func (p *doramalandParser) parseDramaPage(body string) (*DramaInfo, error) {
	info := &DramaInfo{}

	// ── Заголовок ──────────────────────────────────────────────────────────────
	// about-serial-header__title: "Смех в «Вайкики» 1 сезон сериал с 2018 г."
	reTitleBlock := regexp.MustCompile(`class="about-serial-header__title"[^>]*>([^<]+)<`)
	if m := reTitleBlock.FindStringSubmatch(body); len(m) >= 2 {
		title := strings.TrimSpace(m[1])
		// Убираем "сериал с YYYY г." из конца
		title = regexp.MustCompile(`\s+сериал\s+с\s+\d{4}\s+г\.$`).ReplaceAllString(title, "")
		// Убираем "1 сезон", "2 сезон" из конца
		title = regexp.MustCompile(`\s+\d+\s+сезон\s*$`).ReplaceAllString(title, "")
		info.Title = strings.TrimSpace(title)
	}
	if info.Title == "" {
		if t := metaContent(body, "og:title"); t != "" {
			info.Title = stripTags(t)
		}
	}

	// ── Год ────────────────────────────────────────────────────────────────────
	// "Дата выхода: 5 февраля 2018" или "сериал с 2018 г." в заголовке
	reDate := regexp.MustCompile(`(?:Дата\s+выхода|сериал\s+с)[^0-9]*(\d{4})`)
	if m := reDate.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseInt(m[1]); ok && v >= 1990 {
			info.ReleaseYear = ptr(v)
		}
	}

	// ── Статус выпуска ─────────────────────────────────────────────────────────
	// class="about-serial__status completed" → "завершена"
	// class="about-serial__status ongoing"   → "выходит"
	reStatus := regexp.MustCompile(`class="about-serial__status\s+(\w+)"`)
	if m := reStatus.FindStringSubmatch(body); len(m) >= 2 {
		switch m[1] {
		case "completed":
			info.ReleaseTag = "released"
		case "ongoing", "airing":
			info.ReleaseTag = "ongoing"
		case "announced":
			info.ReleaseTag = "planned"
		default:
			info.ReleaseTag = parseReleaseTagFromBody(body)
		}
	} else {
		info.ReleaseTag = parseReleaseTagFromBody(body)
	}

	// ── Перевод ────────────────────────────────────────────────────────────────
	// "С русской озвучкой: Дубляж, SoftBox, Русские субтитры завершена"
	// "завершена" после озвучки = translated, иначе проверяем ещё
	reTranslation := regexp.MustCompile(`(?i)С\s+русской\s+озвучкой[^<]{0,200}завершена`)
	if reTranslation.MatchString(body) {
		info.TranslationTag = "translated"
	} else if strings.Contains(strings.ToLower(body), "с русской озвучкой") {
		info.TranslationTag = "translating"
	} else {
		info.TranslationTag = parseTranslationTagFromBody(body)
	}

	// ── Жанры ─────────────────────────────────────────────────────────────────
	// <span class="serial-genres-links">Жанры: Драма, Комедия, ...</span>
	// или itemprop="genre" content="Драма, Комедия, ..."
	reGenreMeta := regexp.MustCompile(`itemprop="genre"[^>]*content="([^"]+)"`)
	if m := reGenreMeta.FindStringSubmatch(body); len(m) >= 2 {
		for _, g := range strings.Split(m[1], ",") {
			g = strings.TrimSpace(g)
			if g != "" {
				info.Genres = append(info.Genres, g)
			}
		}
	}
	if len(info.Genres) == 0 {
		reGenreBlock := regexp.MustCompile(`serial-genres-links[^>]*>Жанры:\s*([^<]+)<`)
		if m := reGenreBlock.FindStringSubmatch(body); len(m) >= 2 {
			for _, g := range strings.Split(m[1], ",") {
				g = strings.TrimSpace(g)
				if g != "" {
					info.Genres = append(info.Genres, g)
				}
			}
		}
	}

	// ── Страна ────────────────────────────────────────────────────────────────
	// "Страна: Южная Корея"
	reCountry := regexp.MustCompile(`Страна:\s*</[^>]+>\s*<[^>]+>([^<]+)<|Страна:\s*([^\n<,]{2,40})`)
	if m := reCountry.FindStringSubmatch(body); len(m) >= 2 {
		c := strings.TrimSpace(m[1])
		if c == "" {
			c = strings.TrimSpace(m[2])
		}
		info.Country = c
	}
	if info.Country == "" {
		info.Country = parseCountryFromBody(body)
	}

	// ── Рейтинг ───────────────────────────────────────────────────────────────
	// itemprop="ratingValue" content="7.6"
	reRating := regexp.MustCompile(`itemprop="ratingValue"[^>]*content="([0-9.,]+)"`)
	if m := reRating.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseFloat(m[1]); ok && v > 0 {
			info.Rating = ptr(v)
		}
	}

	// ── Длительность ──────────────────────────────────────────────────────────
	// "Длительность: 1 ч. 5 мин." → 65 мин
	reDurHM := regexp.MustCompile(`(?i)Длительность[^0-9]*(\d+)\s*ч[^0-9]*(\d+)\s*мин`)
	reDurM := regexp.MustCompile(`(?i)Длительность[^0-9]*(\d+)\s*мин`)
	if m := reDurHM.FindStringSubmatch(body); len(m) >= 3 {
		h, _ := parseInt(m[1])
		mn, _ := parseInt(m[2])
		total := h*60 + mn
		if total > 0 {
			info.EpisodeDurationMin = ptr(total)
		}
	} else if m := reDurM.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseInt(m[1]); ok && v > 0 {
			info.EpisodeDurationMin = ptr(v)
		}
	}

	// ── Серии ─────────────────────────────────────────────────────────────────
	// "Количество серий: 20" или "Дорама:20 серий"
	reEps := regexp.MustCompile(`(?i)(?:Количество\s+серий|Дорама)[^0-9]*(\d+)\s*серий?`)
	if m := reEps.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseInt(m[1]); ok && v > 0 {
			info.Seasons = []SeasonInfo{{SeasonNumber: 1, EpisodeCount: v}}
		}
	}

	return info, nil
}
