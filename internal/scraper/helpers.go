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
	// Приоритет №1: стандартный видео-метатег video:duration в формате ISO 8601 (напр. "PT45M").
	// Это структурированные данные (OpenGraph video), а не текст рядом с лейблом на человеческом
	// языке — гораздо надёжнее и не зависит от того, как конкретно сайт оформил разметку.
	// Подтверждено вживую на dorama.land: <meta property="video:duration" content="PT45M">.
	if d := metaContent(body, "video:duration"); d != "" {
		if m := regexp.MustCompile(`(?i)PT(\d+)M`).FindStringSubmatch(d); len(m) >= 2 {
			if v, ok := parseInt(m[1]); ok && v > 0 && v < 300 {
				return ptr(v)
			}
		}
	}

	durationPatterns := []string{
		`(?i)(?:длительность|время|duration|продолжительность)[^:]*:\s*(?:</[^>]+>)*\s*(\d+)\s*(?:мин|min)`,
		`(?i)"duration"\s*:\s*"?PT(\d+)M"?`, // ISO 8601 (JSON-LD)
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

// ── Озвучка / студии перевода ─────────────────────────────────────────────────

// parseVoiceoverLabelled ищет список студий озвучки/перевода по лейблу "Озвучка:" —
// формат, подтверждённый вживую на dorama.land: "Озвучка: DubLikTV, Light Breeze, ...".
// Возвращает сырую строку через запятую как есть — сопоставление с фиксированным
// списком VOICEOVER_OPTIONS происходит уже на фронте (там же по каждой из перечисленных
// через запятую студий, а не только по первой).
//
// Значение после лейбла может быть обёрнуто в <a>-ссылку (подтверждено вживую на
// doramy.club: "Озвучка: <a href=...>DubLik-TV</a>") — простой захват [^<\n]+ сразу
// после двоеточия такой случай не ловит (останавливается на открывающем теге <a>) — поэтому
// вместо точечного регекса берём небольшое окно текста после лейбла и чистим теги целиком (stripTags
// уже умеет вычищать произвольные теги внутри).
func parseVoiceoverLabelled(body string) string {
	idx := strings.Index(body, "Озвучка")
	if idx == -1 {
		return ""
	}

	window := body[idx+len("Озвучка"):]
	// Отрезаем перед следующим известным лейблом, если он попадает в окно — чтобы не
	// захватить лишнего (напр. список актёров через "В ролях").
	if stop := regexp.MustCompile(`(?i)В\s+ролях|Режиссёр|Режиссер|Сценарист|Канал`).FindStringIndex(window); stop != nil {
		window = window[:stop[0]]
	}
	if len(window) > 400 {
		window = window[:400]
	}

	// Сам лейбл может иметь окончание типа " и перевод:" — отрезаем всё до первого двоеточия.
	if colonIdx := strings.Index(window, ":"); colonIdx != -1 {
		window = window[colonIdx+1:]
	}

	v := strings.TrimSpace(stripTags(window))
	v = strings.Trim(v, ". ")
	if v == "" {
		return ""
	}
	if len([]rune(v)) > 255 {
		v = string([]rune(v)[:255])
	}
	return v
}

// ── Постер ──────────────────────────────────────────────────────────────────────────────

// parsePosterURL извлекает URL постера из og:image / twitter:image или JSON-LD image.
// Вызывается в конце каждого парсера на body страницы конкретной дорамы (не поиска),
// чтобы не таскать постер через отдельный CORS-прокси на фронте (часто блокируется
// антибот-защитой целевого сайта) — бэкенд уже скачал HTML и может отдать готовый URL картинки.
func parsePosterURL(body string) string {
	if og := metaContent(body, "og:image"); og != "" {
		return resolvePosterURL(og)
	}
	if og := metaContent(body, "og:image:secure_url"); og != "" {
		return resolvePosterURL(og)
	}
	if tw := metaContent(body, "twitter:image"); tw != "" {
		return resolvePosterURL(tw)
	}
	if jl := jsonLDField(body, "image"); jl != "" {
		return resolvePosterURL(jl)
	}
	// <link rel="image_src" href="..."> — старый, но всё ещё встречающийся способ указать постер.
	if m := regexp.MustCompile(`(?i)<link[^>]+rel="image_src"[^>]+href="([^"]+)"`).FindStringSubmatch(body); len(m) >= 2 {
		return resolvePosterURL(m[1])
	}
	// Явно помеченная постером/обложкой картинка (class="poster"/"cover") — на случай, если сайт
	// не проставляет og:image вовсе, но верстает страницу с понятной семантикой класса.
	if m := regexp.MustCompile(`(?i)<img[^>]+class="[^"]*(?:poster|cover)[^"]*"[^>]*\ssrc="([^"]+)"`).FindStringSubmatch(body); len(m) >= 2 {
		return resolvePosterURL(m[1])
	}
	return ""
}

// resolvePosterURL чинит protocol-relative ссылки (//host/img.jpg) до полного https:// URL.
func resolvePosterURL(u string) string {
	u = strings.TrimSpace(u)
	if strings.HasPrefix(u, "//") {
		return "https:" + u
	}
	return u
}
