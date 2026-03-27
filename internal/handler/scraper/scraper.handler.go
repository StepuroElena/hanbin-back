package scraperhandler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hanbin/hanbin-back/internal/scraper"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/dramas/scrape", h.Scrape)
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

	info, err := scraper.Scrape(r.Context(), title, siteURL)
	if err != nil {
		// ErrNotFound: дорама не найдена или сайт недоступен/не поддерживается
		// bad url: невалидный site_url — ошибка клиента
		// В обоих случаях отдаём 404, никогда не 5xx
		writeError(w, http.StatusNotFound, "дорама не найдена на этом сайте", true)
		return
	}

	writeJSON(w, http.StatusOK, info)
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
