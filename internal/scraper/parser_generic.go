package scraper

import (
	"context"
	"regexp"
	"strings"
)

// genericParser — универсальный fallback-парсер.
// Работает с любым сайтом о дорамах через набор эвристик:
// Open Graph, JSON-LD, common class/attr patterns, regex по тексту.
type genericParser struct{}

func (p *genericParser) canHandle(_ string) bool { return true }

func (p *genericParser) parse(_ context.Context, body, _ string) (*DramaInfo, error) {
	info := &DramaInfo{}

	// ── Заголовок ──────────────────────────────────────────────────────────────
	// Приоритет: JSON-LD > og:title > <title>
	if t := jsonLDField(body, "name"); t != "" {
		info.Title = t
	} else if t := metaContent(body, "og:title"); t != "" {
		info.Title = stripTags(t)
	} else {
		t := betweenTags(body, "<title", "</title>")
		t = betweenTags(t, ">", "")
		info.Title = stripTags(t)
	}

	// ── Год ────────────────────────────────────────────────────────────────────
	yearCandidates := []string{
		jsonLDField(body, "dateCreated"),
		jsonLDField(body, "startDate"),
		metaContent(body, "video:release_date"),
		betweenTags(body, `"year">`, "<"),
		betweenTags(body, `class="year"`, "<"),
		betweenTags(body, "Год:", "<"),
		betweenTags(body, "Year:", "<"),
		betweenTags(body, "Release:", "<"),
	}
	for _, c := range yearCandidates {
		if y := firstMatch(reYear, c); y != "" {
			if v, ok := parseInt(y); ok {
				info.ReleaseYear = ptr(v)
				break
			}
		}
	}

	// ── Статус выхода ──────────────────────────────────────────────────────────
	info.ReleaseTag = parseReleaseTagFromBody(body)

	// ── Статус перевода ────────────────────────────────────────────────────────
	info.TranslationTag = parseTranslationTagFromBody(body)

	// ── Жанры ─────────────────────────────────────────────────────────────────
	info.Genres = parseGenresGeneric(body)

	// ── Рейтинг ───────────────────────────────────────────────────────────────
	// JSON-LD имеет приоритет
	if r := jsonLDField(body, "ratingValue"); r != "" {
		if v, ok := parseFloat(r); ok {
			info.Rating = ptr(v)
		}
	}
	if info.Rating == nil {
		info.Rating = parseRatingFromBody(body)
	}

	// ── Страна ────────────────────────────────────────────────────────────────
	if c := jsonLDField(body, "countryOfOrigin"); c != "" {
		info.Country = c
	} else {
		info.Country = parseCountryFromBody(body)
	}

	// ── Длительность ──────────────────────────────────────────────────────────
	if d := jsonLDField(body, "timeRequired"); d != "" {
		// ISO 8601 duration: PT45M
		isoRe := regexp.MustCompile(`PT(\d+)M`)
		if m := isoRe.FindStringSubmatch(d); len(m) >= 2 {
			if v, ok := parseInt(m[1]); ok {
				info.EpisodeDurationMin = ptr(v)
			}
		}
	}
	if info.EpisodeDurationMin == nil {
		info.EpisodeDurationMin = parseDurationFromBody(body)
	}

	// ── Сезоны ─────────────────────────────────────────────────────────────────
	info.Seasons = parseSeasonsGeneric(body)

	return info, nil
}

// parseGenresGeneric ищет жанры в ссылках, JSON-LD и обычном тексте.
func parseGenresGeneric(body string) []string {
	seen := map[string]bool{}
	genres := []string{}

	// JSON-LD: "genre": ["Romance", "Thriller"]
	jsonGenreRe := regexp.MustCompile(`(?i)"genre"\s*:\s*\[([^\]]+)\]`)
	if m := jsonGenreRe.FindStringSubmatch(body); len(m) >= 2 {
		itemRe := regexp.MustCompile(`"([^"]+)"`)
		items := itemRe.FindAllStringSubmatch(m[1], -1)
		for _, it := range items {
			g := strings.TrimSpace(it[1])
			if g != "" && !seen[g] {
				seen[g] = true
				genres = append(genres, g)
			}
		}
	}
	// Одиночный жанр в JSON-LD: "genre": "Romance"
	if len(genres) == 0 {
		if g := jsonLDField(body, "genre"); g != "" {
			genres = append(genres, g)
		}
	}

	// Ссылки вида /genre/, /zhanr/, /tag/, /category/
	if len(genres) == 0 {
		linkRe := regexp.MustCompile(`(?i)/(?:genre|zhanr|tag|category)/[^"?#]+["?#][^>]*>([^<]{2,40})</a>`)
		for _, m := range linkRe.FindAllStringSubmatch(body, -1) {
			g := strings.TrimSpace(m[1])
			if g != "" && !seen[g] && !isNavWord(g) {
				seen[g] = true
				genres = append(genres, g)
			}
		}
	}

	return genres
}

// parseSeasonsGeneric ищет информацию о сезонах и сериях.
func parseSeasonsGeneric(body string) []SeasonInfo {
	// Паттерн "Сезон N — X серий" / "Season N: X episodes"
	seasonRe := regexp.MustCompile(`(?i)(?:Сезон|Season)\s*(\d+)[^\d<]{0,30}?(\d+)\s*(?:эпизод|серий|серии|серия|ep|eps|episode)`)
	matches := seasonRe.FindAllStringSubmatch(body, -1)

	if len(matches) > 0 {
		seasons := make([]SeasonInfo, 0, len(matches))
		for _, m := range matches {
			sn, ok1 := parseInt(m[1])
			ep, ok2 := parseInt(m[2])
			if ok1 && ok2 && ep > 0 {
				seasons = append(seasons, SeasonInfo{SeasonNumber: sn, EpisodeCount: ep})
			}
		}
		if len(seasons) > 0 {
			return seasons
		}
	}

	// Fallback: одно значение количества серий → 1 сезон
	epStr := firstMatch(reEpCount, body)
	if epStr == "" {
		// "16 серий" / "16 ep" без слова "эпизод" перед числом
		epFallbackRe := regexp.MustCompile(`(\d+)\s*(?:серий|серии|серия|ep\b|eps\b)`)
		epStr = firstMatch(epFallbackRe, body)
	}
	if epStr != "" {
		if v, ok := parseInt(epStr); ok && v > 0 && v < 1000 {
			return []SeasonInfo{{SeasonNumber: 1, EpisodeCount: v}}
		}
	}

	return []SeasonInfo{}
}

// isNavWord отфильтровывает общие навигационные слова, которые не являются жанрами.
func isNavWord(s string) bool {
	nav := []string{"главная", "home", "about", "контакты", "search", "поиск", "войти", "login", "register"}
	sl := strings.ToLower(strings.TrimSpace(s))
	for _, n := range nav {
		if sl == n {
			return true
		}
	}
	return false
}
