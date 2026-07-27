package movie

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	domain "github.com/hanbin/hanbin-back/internal/domain/movie"
	"github.com/hanbin/hanbin-back/internal/middleware"
	svc "github.com/hanbin/hanbin-back/internal/service/movie"
)

// Handler обрабатывает HTTP-запросы для фильмов.
type Handler struct {
	service *svc.Service
}

func NewHandler(service *svc.Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes регистрирует маршруты:
//
//	GET  /api/v1/movies        — список фильмов пользователя (требует JWT)
//	POST /api/v1/movies        — добавить фильм (требует JWT)
//	GET  /api/v1/movies/stats  — счётчики просмотрено/запланировано (требует JWT)
//	PATCH /api/v1/movies/{id}  — изменить статус просмотра (требует JWT)
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/movies", middleware.Auth(http.HandlerFunc(h.handleCollection)))
	// ВАЖНО: регистрируется ДО "/api/v1/movies/" — точный паттерн в Go 1.22+ mux
	// имеет приоритет, поэтому "stats" не попадёт в handleItem как id (см. drama.handler.go).
	mux.Handle("/api/v1/movies/stats", middleware.Auth(http.HandlerFunc(h.Stats)))
	mux.Handle("/api/v1/movies/", middleware.Auth(http.HandlerFunc(h.handleItem)))
}

// handleCollection — диспетчер для /api/v1/movies
func (h *Handler) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleItem — диспетчер для /api/v1/movies/{id}...
func (h *Handler) handleItem(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/movies/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 1 {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "invalid movie id")
			return
		}
		switch r.Method {
		case http.MethodPatch:
			h.UpdateStatus(w, r, id)
		case http.MethodDelete:
			h.Delete(w, r, id)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// PATCH /api/v1/movies/{id}/archive или /unarchive
	if len(parts) == 2 && (parts[1] == "archive" || parts[1] == "unarchive") {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "invalid movie id")
			return
		}
		if r.Method != http.MethodPatch {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		isArchived := parts[1] == "archive"
		h.SetArchived(w, r, id, isArchived)
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

// List godoc
//
//	GET /api/v1/movies
//	Header: Authorization: Bearer <token>
//	200 OK  → []MovieOutput (JSON)
//	401 Unauthorized
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	out, err := h.service.GetAllByProfileID(r.Context(), profileID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Create godoc
//
//	POST /api/v1/movies
//	Header: Authorization: Bearer <token>
//	Body: CreateInput (JSON)
//	201 Created  → MovieOutput (JSON)
//	400 Bad Request
//	401 Unauthorized
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body svc.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	out, err := h.service.Create(r.Context(), profileID, body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// Stats godoc
//
//	GET /api/v1/movies/stats
//	Header: Authorization: Bearer <token>
//	200 OK  → StatsOutput (JSON: { movies_watched, movies_planned })
//	401 Unauthorized
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	out, err := h.service.GetStats(r.Context(), profileID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// UpdateStatus godoc
//
//	PATCH /api/v1/movies/{id}
//	Header: Authorization: Bearer <token>
//	Body: UpdateStatusInput (JSON: { watch_status: "planned"|"watched" })
//	200 OK  → MovieOutput (JSON)
//	400 Bad Request
//	401 Unauthorized
//	404 Not Found
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request, movieID int64) {
	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body svc.UpdateStatusInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	out, err := h.service.UpdateStatus(r.Context(), profileID, movieID, body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// SetArchived godoc
//
//	PATCH /api/v1/movies/{id}/archive   — архивировать фильм
//	PATCH /api/v1/movies/{id}/unarchive — вернуть из архива
//	Header: Authorization: Bearer <token>
//	200 OK  → MovieOutput (JSON)
//	401 Unauthorized
//	404 Not Found
func (h *Handler) SetArchived(w http.ResponseWriter, r *http.Request, movieID int64, isArchived bool) {
	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	out, err := h.service.SetArchived(r.Context(), profileID, movieID, isArchived)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Delete godoc
//
//	DELETE /api/v1/movies/{id}
//	Header: Authorization: Bearer <token>
//	204 No Content  — фильм удалён
//	400 Bad Request — фильм не архивирован
//	401 Unauthorized
//	404 Not Found
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, movieID int64) {
	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.Delete(r.Context(), profileID, movieID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "movie not found")
	case errors.Is(err, domain.ErrTitleRequired),
		errors.Is(err, domain.ErrTitleTooLong),
		errors.Is(err, domain.ErrGenreRequired),
		errors.Is(err, domain.ErrGenreTooLong),
		errors.Is(err, domain.ErrCountryTooLong),
		errors.Is(err, domain.ErrInvalidYear),
		errors.Is(err, domain.ErrInvalidWatchStatus),
		errors.Is(err, domain.ErrNotArchived),
		errors.Is(err, domain.ErrProfileIDRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
