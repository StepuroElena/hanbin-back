package random

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/hanbin/hanbin-back/internal/middleware"
	svc "github.com/hanbin/hanbin-back/internal/service/random"
)

// Handler обрабатывает HTTP-запросы фичи «Рандом».
type Handler struct {
	service *svc.Service
}

func NewHandler(service *svc.Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes регистрирует маршруты:
//
//	GET /api/v1/random/facets?type=any|movie|series                                        — требует JWT
//	GET /api/v1/random/pick?type=any|movie|series&genre=<g>&exclude_kind=<k>&exclude_id=<id> — требует JWT
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/random/facets", middleware.Auth(http.HandlerFunc(h.Facets)))
	mux.Handle("/api/v1/random/pick", middleware.Auth(http.HandlerFunc(h.Pick)))
}

// Facets godoc
//
//	GET /api/v1/random/facets?type=any|movie|series
//	Header: Authorization: Bearer <token>
//	200 OK → FacetsOutput (JSON: { genres, has_movies, has_series })
//	401 Unauthorized
func (h *Handler) Facets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	typeFilter := svc.ParseType(r.URL.Query().Get("type"))

	out, err := h.service.GetFacets(r.Context(), profileID, typeFilter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Pick godoc
//
//	GET /api/v1/random/pick?type=any|movie|series&genre=<жанр>&exclude_kind=drama|movie&exclude_id=<id>
//	Header: Authorization: Bearer <token>
//	200 OK       → PickOutput (JSON)
//	404 Not Found — под фильтры ничего не нашлось (пустой пул "Запланировано")
//	401 Unauthorized
//
// genre/exclude_kind/exclude_id опциональны: genre="" — без фильтра по жанру,
// exclude_id=0 или отсутствует — без исключения (первый подбор, не реролл).
func (h *Handler) Pick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	q := r.URL.Query()
	typeFilter := svc.ParseType(q.Get("type"))
	genre := q.Get("genre")

	var excludeKind svc.Kind
	switch q.Get("exclude_kind") {
	case "drama":
		excludeKind = svc.KindDrama
	case "movie":
		excludeKind = svc.KindMovie
	}

	excludeID, _ := strconv.ParseInt(q.Get("exclude_id"), 10, 64) // ошибка парсинга = 0 = без исключения, это ок

	out, err := h.service.Pick(r.Context(), profileID, typeFilter, genre, excludeKind, excludeID)
	if err != nil {
		if errors.Is(err, svc.ErrNoMatch) {
			writeError(w, http.StatusNotFound, "no matching planned title")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ── helpers ───────────────────────────────────────────────────────────────────

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
