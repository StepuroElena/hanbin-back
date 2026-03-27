package scraper

import (
	"regexp"
	"strings"
)

// ── Статус выхода ─────────────────────────────────────────────────────────────

// parseReleaseTagFromBody определяет статус выхода дорамы из HTML-тела страницы.
// Возвращает "ongoing", "planned" или "released".
func parseReleaseTagFromBody(body string) string {
	lower := strings.ToLower(body)

	ongoingKeywords := []string{
		"онгоинг", "ongoing", "в эфире", "выходит", "airing", "currently airing",
		"в производстве", "продолжается", "новые серии",
	}
	for _, kw := range ongoingKeywords {
		if strings.Contains(lower, kw) {
			return "ongoing"
		}
	}

	plannedKeywords := []string{
		"анонс", "анонсирован", "запланирован", "upcoming", "announced", "coming soon",
		"скоро", "ожидается",
	}
	for _, kw := range plannedKeywords {
		if strings.Contains(lower, kw) {
			return "planned"
		}
	}

	releasedKeywords := []string{
		"завершён", "завершен", "завершена", "completed", "finished", "released",
		"вышел", "вышла", "вышло", "aired", "ended",
	}
	for _, kw := range releasedKeywords {
		if strings.Contains(lower, kw) {
			return "released"
		}
	}

	return "released" // разумный default
}

// ── Статус перевода ───────────────────────────────────────────────────────────

// parseTranslationTagFromBody определяет статус перевода из текста страницы.
func parseTranslationTagFromBody(body string) string {
	lower := strings.ToLower(body)

	translatingKeywords := []string{
		"переводится", "переводим", "в переводе", "идёт перевод", "идет перевод",
		"translating", "translation in progress", "субтитры в процессе",
	}
	for _, kw := range translatingKeywords {
		if strings.Contains(lower, kw) {
			return "translating"
		}
	}

	translatedKeywords := []string{
		"переведено", "перевод завершён", "перевод завершен", "fully translated",
		"translated", "субтитры готовы", "озвучка готова",
	}
	for _, kw := range translatedKeywords {
		if strings.Contains(lower, kw) {
			return "translated"
		}
	}

	return "" // сайт не даёт информации о переводе
}

// ── Страна ────────────────────────────────────────────────────────────────────

var countryMap = map[string]string{
	// Корея
	"корея": "Корея", "южная корея": "Корея", "korea": "Корея", "south korea": "Корея",
	"korean": "Корея", "k-drama": "Корея",
	// Китай
	"китай": "Китай", "китайская": "Китай", "china": "Китай", "chinese": "Китай", "c-drama": "Китай",
	// Япония
	"япония": "Япония", "japan": "Япония", "japanese": "Япония", "j-drama": "Япония",
	// Тайвань
	"тайвань": "Тайвань", "taiwan": "Тайвань", "taiwanese": "Тайвань",
	// Таиланд
	"таиланд": "Таиланд", "thailand": "Таиланд", "thai": "Таиланд",
}

// parseCountryFromBody пытается определить страну производства из текста страницы.
func parseCountryFromBody(body string) string {
	lower := strings.ToLower(body)

	// Ищем явное указание страны в атрибутах/тексте
	countryPatterns := []string{
		`(?i)страна[^:]*:\s*</[^>]+>\s*<[^>]+>([^<]+)<`,
		`(?i)country[^:]*:\s*</[^>]+>\s*<[^>]+>([^<]+)<`,
		`(?i)страна[^:]*:\s*([^\n<,]+)`,
		`(?i)country[^:]*:\s*([^\n<,]+)`,
		`(?i)"countryOfOrigin"\s*:\s*"([^"]+)"`,
	}
	for _, pat := range countryPatterns {
		re := regexp.MustCompile(pat)
		if m := re.FindStringSubmatch(body); len(m) >= 2 {
			candidate := strings.ToLower(strings.TrimSpace(m[1]))
			for k, v := range countryMap {
				if strings.Contains(candidate, k) {
					return v
				}
			}
		}
	}

	// Грубый поиск по ключевым словам в тексте
	for k, v := range countryMap {
		if strings.Contains(lower, k) {
			return v
		}
	}

	return ""
}

// ── Рейтинг ───────────────────────────────────────────────────────────────────

// parseRatingFromBody извлекает числовой рейтинг из текста страницы.
func parseRatingFromBody(body string) *float64 {
	// Ищем в специфичных местах: itemprop, class="rating", class="score"
	ratingPatterns := []string{
		`(?i)itemprop="ratingValue"[^>]*>([0-9.,]+)`,
		`(?i)class="[^"]*(?:rating|score|оценка)[^"]*"[^>]*>([0-9.,]+)`,
		`(?i)"ratingValue"\s*:\s*"?([0-9.,]+)"?`,
		`(?i)Оценка[^:]*:\s*([0-9.,]+)`,
		`(?i)Rating[^:]*:\s*([0-9.,]+)`,
	}
	for _, pat := range ratingPatterns {
		re := regexp.MustCompile(pat)
		if m := re.FindStringSubmatch(body); len(m) >= 2 {
			if v, ok := parseFloat(m[1]); ok && v > 0 && v <= 10 {
				return ptr(v)
			}
		}
	}
	return nil
}

// ── Длительность эпизода ─────────────────────────────────────────────────────

// parseDurationFromBody ищет продолжительность одного эпизода в минутах.
func parseDurationFromBody(body string) *int {
	durationPatterns := []string{
		`(?i)(?:длительность|duration|продолжительность)[^:]*:\s*(?:</[^>]+>)*\s*(\d+)\s*(?:мин|min)`,
		`(?i)"duration"\s*:\s*"?PT(\d+)M"?`, // ISO 8601
		`(?i)(\d+)\s*(?:мин(?:ут)?\.?|min\.?)\s*(?:/\s*(?:эп|ep))`,
		`(?i)(\d+)\s*minutes?\s*per\s*episode`,
	}
	for _, pat := range durationPatterns {
		re := regexp.MustCompile(pat)
		if m := re.FindStringSubmatch(body); len(m) >= 2 {
			if v, ok := parseInt(m[1]); ok && v > 0 && v < 300 {
				return ptr(v)
			}
		}
	}

	// Последний шанс: общий reEpCount ищет просто число с "мин"
	if s := firstMatch(reDuration, body); s != "" {
		if v, ok := parseInt(s); ok && v > 0 && v < 300 {
			return ptr(v)
		}
	}

	return nil
}
