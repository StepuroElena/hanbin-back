// Package scraper реализует парсинг информации о дорамах с внешних сайтов.
package scraper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ── Выходная структура ────────────────────────────────────────────────────────

type SeasonInfo struct {
	SeasonNumber int    `json:"season_number"`
	EpisodeCount int    `json:"episode_count"`
	Title        string `json:"title,omitempty"`
}

type DramaInfo struct {
	Title              string       `json:"title"`
	ReleaseYear        *int         `json:"release_year"`
	ReleaseTag         string       `json:"release_tag"`
	TranslationTag     string       `json:"translation_tag"`
	Genres             []string     `json:"genres"`
	Rating             *float64     `json:"rating"`
	Country            string       `json:"country"`
	EpisodeDurationMin *int         `json:"episode_duration_min"`
	Seasons            []SeasonInfo `json:"seasons"`
	SourceURL          string       `json:"source_url"`
}

// ── Интерфейс парсера ─────────────────────────────────────────────────────────

type parser interface {
	canHandle(host string) bool
	parse(ctx context.Context, body string, rawURL string) (*DramaInfo, error)
}

var parsers = []parser{
	&doramatvParser{},
	&doramalandParser{},
	&mydramalistParser{},
	&shikimoriParser{},
	&genericParser{},
}

// ── Публичная функция ─────────────────────────────────────────────────────────

func Scrape(ctx context.Context, title, siteURL string) (*DramaInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	normalized, err := normalizeURL(siteURL, title)
	if err != nil {
		// Невалидный URL — клиентская ошибка, пробрасываем как есть (400 на уровне хендлера)
		return nil, fmt.Errorf("scraper: bad url: %w", err)
	}

	body, finalURL, err := fetch(ctx, normalized)
	if err != nil {
		return nil, ErrNotFound
	}

	host := extractHost(finalURL)

	for _, p := range parsers {
		if p.canHandle(host) {
			info, err := p.parse(ctx, body, finalURL)
			if err != nil {
				// Парсер вернул ErrNotFound — пробрасываем как есть
				if errors.Is(err, ErrNotFound) {
					return nil, ErrNotFound
				}
				// Любая другая ошибка парсера — тоже «не нашли», не 5xx
				return nil, ErrNotFound
			}
			if info.Title == "" {
				info.Title = title
			}
			// Парсер может выставить точный URL страницы дорамы — не перезаписываем
			if info.SourceURL == "" {
				info.SourceURL = finalURL
			}
			return info, nil
		}
	}

	// Нет парсера для этого хоста — сайт не поддерживается, не 5xx
	return nil, ErrNotFound
}

// ── HTTP fetch ────────────────────────────────────────────────────────────────

func fetch(ctx context.Context, rawURL string) (body string, finalURL string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru,en;q=0.9")
	// Не запрашиваем gzip — Go читает тело как есть, без декодирования
	// req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	client := &http.Client{
		Timeout: 14 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
	}

	limited := io.LimitReader(resp.Body, 2*1024*1024)
	b, err := io.ReadAll(limited)
	if err != nil {
		return "", "", err
	}
	return string(b), resp.Request.URL.String(), nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// normalizeURL строит корректный URL для запроса.
// Ключевая логика: для doramatv ВСЕГДА идём через /search,
// даже если передан путь типа /serial/... — это старый формат, он не работает.
func normalizeURL(rawURL, title string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if !strings.HasPrefix(rawURL, "http") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	host := strings.ToLower(u.Host)

	switch {
	case strings.Contains(host, "doramatv"):
		// Всегда используем поиск — прямые slug-URL на этом сайте ненадёжны
		u.Path = "/search"
		q := url.Values{}
		q.Set("q", title)
		u.RawQuery = q.Encode()

	case strings.Contains(host, "dorama.land"),
		strings.Contains(host, "doramy.club"),
		strings.Contains(host, "doramy.info"),
		strings.Contains(host, "doram-ru"),
		strings.Contains(host, "dorama24"),
		strings.Contains(host, "mydramalist"):
		if u.Path == "" || u.Path == "/" {
			u.Path = "/search"
			q := url.Values{}
			q.Set("q", title)
			u.RawQuery = q.Encode()
		}

	default:
		// Для остальных сайтов: если путь не задан — пробуем slug
		if u.Path == "" || u.Path == "/" {
			u.Path = "/" + titleToSlug(title)
		}
	}

	return u.String(), nil
}

func extractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}

func titleToSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, slug)
	re := regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}

// ── Общие regexp-хелперы ──────────────────────────────────────────────────────

var (
	reYear     = regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)
	reEpCount  = regexp.MustCompile(`(\d+)\s*(?:эпизод|episode|серий|серия|серии|ep|eps)`)
	reDuration = regexp.MustCompile(`(\d+)\s*(?:мин|min|минут)`)
	reRating   = regexp.MustCompile(`(\d+[.,]\d+)`)
)

func firstMatch(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func parseInt(s string) (int, bool) {
	s = strings.TrimSpace(s)
	v, err := strconv.Atoi(s)
	return v, err == nil
}

func parseFloat(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

func betweenTags(html, openTag, closeTag string) string {
	start := strings.Index(html, openTag)
	if start == -1 {
		return ""
	}
	start += len(openTag)
	end := strings.Index(html[start:], closeTag)
	if end == -1 {
		return ""
	}
	return html[start : start+end]
}

func stripTags(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	s = re.ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func allMatches(re *regexp.Regexp, s string) []string {
	matches := re.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			out = append(out, m[1])
		}
	}
	return out
}

func metaContent(html, key string) string {
	patterns := []string{
		`name="` + key + `"[^>]*content="([^"]*)"`,
		`content="([^"]*)"[^>]*name="` + key + `"`,
		`property="` + key + `"[^>]*content="([^"]*)"`,
		`content="([^"]*)"[^>]*property="` + key + `"`,
	}
	for _, pat := range patterns {
		re := regexp.MustCompile(`(?i)` + pat)
		if m := re.FindStringSubmatch(html); len(m) >= 2 {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

func jsonLDField(html, field string) string {
	re := regexp.MustCompile(`(?i)"` + regexp.QuoteMeta(field) + `"\s*:\s*"([^"]+)"`)
	if m := re.FindStringSubmatch(html); len(m) >= 2 {
		return m[1]
	}
	return ""
}

func ptr[T any](v T) *T { return &v }
