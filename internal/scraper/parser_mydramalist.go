package scraper

import (
	"context"
	"regexp"
	"strings"
)

// mydramalistParser — парсер для mydramalist.com
type mydramalistParser struct{}

func (p *mydramalistParser) canHandle(host string) bool {
	return strings.Contains(host, "mydramalist")
}

func (p *mydramalistParser) parse(_ context.Context, body, _ string) (*DramaInfo, error) {
	info := &DramaInfo{}

	// ── Заголовок ──────────────────────────────────────────────────────────────
	if t := metaContent(body, "og:title"); t != "" {
		// MDL добавляет суффикс " - MyDramaList"
		t = strings.TrimSuffix(strings.TrimSpace(t), " - MyDramaList")
		info.Title = t
	}

	// ── Год ────────────────────────────────────────────────────────────────────
	// MDL хранит год в блоке <span class="year">2024</span> или в meta
	yearRe := regexp.MustCompile(`(?i)class="[^"]*year[^"]*"[^>]*>.*?(\d{4})`)
	if m := yearRe.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseInt(m[1]); ok {
			info.ReleaseYear = ptr(v)
		}
	}
	if info.ReleaseYear == nil {
		if y := firstMatch(reYear, body); y != "" {
			if v, ok := parseInt(y); ok {
				info.ReleaseYear = ptr(v)
			}
		}
	}

	// ── Статус ─────────────────────────────────────────────────────────────────
	info.ReleaseTag = parseReleaseTagFromBody(body)
	info.TranslationTag = "" // MDL не даёт инфу о переводе

	// ── Жанры ─────────────────────────────────────────────────────────────────
	genreRe := regexp.MustCompile(`(?i)/tag/[^"]+">([^<]+)</a>`)
	matches := genreRe.FindAllStringSubmatch(body, -1)
	seen := map[string]bool{}
	for _, m := range matches {
		g := strings.TrimSpace(m[1])
		if g != "" && !seen[g] {
			seen[g] = true
			info.Genres = append(info.Genres, g)
		}
	}

	// ── Рейтинг ───────────────────────────────────────────────────────────────
	// MDL: <span itemprop="ratingValue">8.5</span>
	ratingRe := regexp.MustCompile(`(?i)itemprop="ratingValue"[^>]*>([0-9.,]+)`)
	if m := ratingRe.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseFloat(m[1]); ok {
			info.Rating = ptr(v)
		}
	}
	if info.Rating == nil {
		info.Rating = parseRatingFromBody(body)
	}

	// ── Страна ────────────────────────────────────────────────────────────────
	// MDL: "Country: <a>South Korea</a>"
	countryRe := regexp.MustCompile(`(?i)Country:\s*</[^>]+>\s*<[^>]+>([^<]+)<`)
	if m := countryRe.FindStringSubmatch(body); len(m) >= 2 {
		info.Country = strings.TrimSpace(m[1])
	}
	if info.Country == "" {
		info.Country = parseCountryFromBody(body)
	}

	// ── Длительность ──────────────────────────────────────────────────────────
	// MDL: "Duration: 60 min."
	durRe := regexp.MustCompile(`(?i)Duration:\s*</[^>]+>.*?(\d+)\s*min`)
	if m := durRe.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseInt(m[1]); ok {
			info.EpisodeDurationMin = ptr(v)
		}
	}
	if info.EpisodeDurationMin == nil {
		info.EpisodeDurationMin = parseDurationFromBody(body)
	}

	// ── Сезоны ─────────────────────────────────────────────────────────────────
	// MDL: Episodes: 16 (обычно 1 сезон)
	epRe := regexp.MustCompile(`(?i)Episodes:\s*</[^>]+>.*?(\d+)`)
	if m := epRe.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseInt(m[1]); ok && v > 0 {
			info.Seasons = []SeasonInfo{{SeasonNumber: 1, EpisodeCount: v}}
		}
	}

	return info, nil
}
