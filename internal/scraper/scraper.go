// Package scraper реализует парсинг информации о дорамах с внешних сайтов.
package scraper

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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
	Voiceover          string       `json:"voiceover"`
	Seasons            []SeasonInfo `json:"seasons"`
	SourceURL          string       `json:"source_url"`
	PosterURL          string       `json:"poster_url"`
}

// ── Интерфейс парсера ─────────────────────────────────────────────────────────

type parser interface {
	canHandle(host string) bool
	parse(ctx context.Context, body string, rawURL string) (*DramaInfo, error)
}

var parsers = []parser{
	&doramatvParser{},
	&doramalandParser{},
	&dleDoramaParser{hostMatch: "doramy.club", baseURL: "https://doramy.club"},
	&dleDoramaParser{hostMatch: "doramy.info", baseURL: "https://doramy.info"},
	&dleDoramaParser{hostMatch: "doram-ru", baseURL: "https://doram-ru.org"},
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
		// ВРЕМЕННАЯ ДИАГНОСТИКА: раньше реальная причина (HTTP-статус, таймаут,
		// too many redirects и т.п.) отбрасывалась молча — снаружи было не отличить
		// "сайт реально не знает дораму" от "сайт зарезал запрос с IP Render".
		log.Printf("[scraper] fetch failed for %q (title=%q): %v", normalized, title, err)
		return nil, ErrNotFound
	}

	host := extractHost(finalURL)

	for _, p := range parsers {
		if p.canHandle(host) {
			info, err := p.parse(ctx, body, finalURL)
			if err != nil {
				// Парсер вернул ErrNotFound — пробрасываем как есть
				if errors.Is(err, ErrNotFound) {
					log.Printf("[scraper] parser for host %q returned not-found (title=%q, url=%q)", host, title, finalURL)
					logChallengeMechanics(body)
					return nil, ErrNotFound
				}
				// Любая другая ошибка парсера — тоже «не нашли», не 5xx
				log.Printf("[scraper] parser for host %q errored (title=%q, url=%q): %v", host, title, finalURL, err)
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
		strings.Contains(host, "mydramalist"):
		if u.Path == "" || u.Path == "/" {
			u.Path = "/search"
			q := url.Values{}
			q.Set("q", title)
			u.RawQuery = q.Encode()
		}

	case strings.Contains(host, "doramy.club"),
		strings.Contains(host, "doramy.info"),
		strings.Contains(host, "doram-ru"):
		// Подтверждено вживую на doramy.club (через DevTools в браузере): поиск тут не /search?q=,
		// а корневой /?s= — станартный параметр поиска WordPress. Раньше здесь был выдуманный
		// /search?q=, который на самом деле не существует — из-за этого ни одна дорама на этих
		// сайтах не находилась вообще (страница поиска всегда была пустой/чужой страницей).
		if u.Path == "" || u.Path == "/" {
			u.Path = "/"
			q := url.Values{}
			q.Set("s", title)
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

// snippet — вспомогалка только для временных диагностических логов: обрезает тело
// страницы до length символов и схлопывает переводы строк, чтобы одна запись лога
// была компактной однострочной записью, а не выводом на тысячи строк.
func snippet(s string, length int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) > length {
		return string([]rune(s)[:length]) + "…"
	}
	return s
}

// logChallengeMechanics — ВРЕМЕННАЯ ДИАГНОСТИКА. Страница-источник иногда отдаёт вместо реальной
// страницы антибот-заглушку с JS-челленджем/редиректом. Обычный snippet(body, N) бесполезен,
// если в начале тела лежит огромная base64-картинка спиннера — она съедает весь лимит. Вместо
// этого ищем ключевые слова механизма редиректа/проверки по всему телу и печатаем окно
// вокруг каждого совпадения.
func logChallengeMechanics(body string) {
	keywords := []string{"http-equiv=\"refresh\"", "location.href", "location.replace", "location.reload", "document.cookie", "setTimeout", "window.location", "noindex"}
	found := false
	for _, kw := range keywords {
		idx := strings.Index(strings.ToLower(body), strings.ToLower(kw))
		if idx == -1 {
			continue
		}
		found = true
		start := idx - 150
		if start < 0 {
			start = 0
		}
		end := idx + 300
		if end > len(body) {
			end = len(body)
		}
		log.Printf("[scraper] challenge keyword %q found at %d: %s", kw, idx, snippet(body[start:end], 450))
	}
	if !found {
		log.Printf("[scraper] no known challenge keywords found; body len=%d, last 1000 chars: %s", len(body), snippet(lastN(body, 1000), 1000))
	}
}

// lastN возвращает последние n символов строки (или всю строку, если она короче).
func lastN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
