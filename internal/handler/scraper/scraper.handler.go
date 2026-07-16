package scraperhandler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	scrapersvc "github.com/hanbin/hanbin-back/internal/service/scraper"
)

type Handler struct {
	svc *scrapersvc.Service
}

// NewHandler создаёт хендлер скрейпинга. svc инкапсулирует гибридную
// cache-aside/live логику — хендлер об этом ничего не знает.
func NewHandler(svc *scrapersvc.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/dramas/scrape", h.Scrape)
	mux.HandleFunc("GET /api/v1/dramas/hot", h.Hot)
	mux.HandleFunc("GET /api/v1/dramas/poster-proxy", h.PosterProxy)
}

// Scrape godoc
//
//	GET /api/v1/dramas/scrape?title=My+Demon&site_url=https://m.doramatv.one
//	200 OK        → DramaInfo (JSON)
//	400 Bad Request
//	404 Not Found → { "error": "...", "not_found": true }
//	502 Bad Gateway
func (h *Handler) Scrape(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.URL.Query().Get("title"))
	siteURL := strings.TrimSpace(r.URL.Query().Get("site_url"))

	if title == "" {
		writeError(w, http.StatusBadRequest, "query param 'title' is required", false)
		return
	}
	if siteURL == "" {
		writeError(w, http.StatusBadRequest, "query param 'site_url' is required", false)
		return
	}

	info, err := h.svc.Scrape(r.Context(), title, siteURL)
	if err != nil {
		// ErrNotFound: дорама не найдена или сайт недоступен/не поддерживается
		// bad url: невалидный site_url — ошибка клиента
		// В обоих случаях отдаём 404, никогда не 5xx
		writeError(w, http.StatusNotFound, "дорама не найдена на этом сайте", true)
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// Hot godoc
//
//	GET /api/v1/dramas/hot?limit=10
//	200 OK        → [] scraper.HotDrama (всегда массив, даже пустой [])
//
// Публичный эндпоинт, без авторизации — используется на гостевой странице (блок “Тебе понравится”).
// Никогда не возвращает ошибку клиенту при сбое — отдаёт пустой массив, чтобы фронт просто скрыл блок.
func (h *Handler) Hot(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 30 {
			limit = v
		}
	}

	list, err := h.svc.ScrapeHot(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	writeJSON(w, http.StatusOK, list)
}

// PosterProxy godoc
//
//	GET /api/v1/dramas/poster-proxy?url=https://cdn.example.com/poster.jpg
//	200 OK  → image bytes, Content-Type проксируется от источника
//	400 Bad Request
//	502 Bad Gateway
//
// Зачем нужен: сайты-источники (doramatv и т.п.) часто защищают CDN с картинками
// hotlink-защитой (проверяют Referer/User-Agent) — прямой <img src="..."> из
// браузера на localhost получает 403, хотя тот же URL прекрасно открывается
// с самого сайта. Бэкенд уже умеет ходить на эти сайты (спуфит UA для HTML-скрейпа),
// так что картинку просто стримим через себя тем же способом — фронт обращается
// к нашему домену, а не напрямую к чужому CDN.
func (h *Handler) PosterProxy(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	if raw == "" {
		writeError(w, http.StatusBadRequest, "query param 'url' is required", false)
		return
	}

	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		writeError(w, http.StatusBadRequest, "invalid 'url'", false)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, "не удалось запросить постер", false)
		return
	}

	// Те же заголовки, что и при HTML-скрейпе — некоторые CDN проверяют Referer
	// и требуют, чтобы он совпадал по хосту с картинкой (или с сайтом-источником).
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", u.Scheme+"://"+u.Host+"/")
	req.Header.Set("Accept-Language", "ru,en;q=0.9")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "не удалось загрузить постер", false)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		writeError(w, http.StatusBadGateway, "источник вернул ошибку", false)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		// Источник вернул не картинку (например, HTML-страницу с ошибкой) — не проксируем.
		writeError(w, http.StatusBadGateway, "источник вернул не изображение", false)
		return
	}

	// Ограничиваем размер, чтобы не превратить прокси в вектор для DoS/абьюза.
	limited := io.LimitReader(resp.Body, 8*1024*1024)

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400") // постеры почти не меняются — кешируем на сутки в браузере
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, limited)
}

// ── helpers ───────────────────────────────────────────────────────────────────

type errorResponse struct {
	Error    string `json:"error"`
	NotFound bool   `json:"not_found,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string, notFound bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: msg, NotFound: notFound})
}
