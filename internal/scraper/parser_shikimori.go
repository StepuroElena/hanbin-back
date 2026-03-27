package scraper

import (
	"context"
	"regexp"
	"strings"
)

// shikimoriParser — парсер для shikimori.one (японские дорамы / аниме-дорамы).
type shikimoriParser struct{}

func (p *shikimoriParser) canHandle(host string) bool {
	return strings.Contains(host, "shikimori")
}

func (p *shikimoriParser) parse(_ context.Context, body, _ string) (*DramaInfo, error) {
	info := &DramaInfo{}

	// ── Заголовок ──────────────────────────────────────────────────────────────
	if t := metaContent(body, "og:title"); t != "" {
		info.Title = strings.TrimSpace(t)
	}

	// ── Год ────────────────────────────────────────────────────────────────────
	// Shikimori: <div class="value">2023</div> после "Год"
	yearRe := regexp.MustCompile(`(?i)Год[^<]*</[^>]+>.*?<[^>]+>.*?(\d{4})`)
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
	info.TranslationTag = ""

	// ── Жанры ─────────────────────────────────────────────────────────────────
	genreRe := regexp.MustCompile(`(?i)/genres/[^"]+">([^<]+)</a>`)
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
	ratingRe := regexp.MustCompile(`(?i)class="[^"]*score[^"]*"[^>]*>([0-9.,]+)`)
	if m := ratingRe.FindStringSubmatch(body); len(m) >= 2 {
		if v, ok := parseFloat(m[1]); ok && v > 0 {
			info.Rating = ptr(v)
		}
	}
	if info.Rating == nil {
		info.Rating = parseRatingFromBody(body)
	}

	// ── Страна ─────────────────────────────────────────────────────────────────
	info.Country = "Япония" // shikimori — исключительно японский контент
	if c := parseCountryFromBody(body); c != "" {
		info.Country = c
	}

	// ── Длительность ──────────────────────────────────────────────────────────
	info.EpisodeDurationMin = parseDurationFromBody(body)

	// ── Сезоны/серии ─────────────────────────────────────────────────────────
	epStr := firstMatch(reEpCount, body)
	if epStr != "" {
		if v, ok := parseInt(epStr); ok && v > 0 {
			info.Seasons = []SeasonInfo{{SeasonNumber: 1, EpisodeCount: v}}
		}
	}

	return info, nil
}
