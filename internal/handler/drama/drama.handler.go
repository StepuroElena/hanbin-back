package drama

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	domain "github.com/hanbin/hanbin-back/internal/domain/drama"
	"github.com/hanbin/hanbin-back/internal/middleware"
	svc "github.com/hanbin/hanbin-back/internal/service/drama"
)

// Handler обрабатывает HTTP-запросы для дорам.
type Handler struct {
	service *svc.Service
}

func NewHandler(service *svc.Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes регистрирует маршруты:
//
//	POST   /api/v1/dramas             — добавить дораму (требует JWT)
//	PATCH  /api/v1/dramas/{id}/archive — установить/снять архив (требует JWT)
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/dramas", middleware.Auth(http.HandlerFunc(h.handleCollection)))
	mux.Handle("/api/v1/dramas/", middleware.Auth(http.HandlerFunc(h.handleItem)))
}

// handleCollection — диспетчер для /api/v1/dramas
func (h *Handler) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.Create(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleItem — диспетчер для /api/v1/dramas/{id}/...
func (h *Handler) handleItem(w http.ResponseWriter, r *http.Request) {
	// Ожидаем путь вида /api/v1/dramas/{id}/archive
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/dramas/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 2 && parts[1] == "archive" {
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "invalid drama id")
			return
		}
		if r.Method != http.MethodPatch {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.SetArchived(w, r, id)
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

// Create godoc
//
//	POST /api/v1/dramas
//	Header: Authorization: Bearer <token>
//	Body: CreateInput (JSON)
//	201 Created  → DramaOutput (JSON)
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

// SetArchived godoc
//
//	PATCH /api/v1/dramas/{id}/archive
//	Header: Authorization: Bearer <token>
//	Body: {"is_archived": true}
//	200 OK  → DramaOutput (JSON)
//	400 Bad Request
//	401 Unauthorized
//	404 Not Found
func (h *Handler) SetArchived(w http.ResponseWriter, r *http.Request, dramaID int64) {
	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body svc.ArchiveInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	out, err := h.service.SetArchived(r.Context(), profileID, dramaID, body.IsArchived)
	if err != nil {
		writeServiceError(w, err)
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

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "drama not found")
	case errors.Is(err, domain.ErrTitleRequired),
		errors.Is(err, domain.ErrWatchURLRequired),
		errors.Is(err, domain.ErrGenreRequired),
		errors.Is(err, domain.ErrCountryRequired),
		errors.Is(err, domain.ErrTitleTooLong),
		errors.Is(err, domain.ErrGenreTooLong),
		errors.Is(err, domain.ErrCountryTooLong),
		errors.Is(err, domain.ErrInvalidYear),
		errors.Is(err, domain.ErrInvalidRating),
		errors.Is(err, domain.ErrInvalidReleaseTag),
		errors.Is(err, domain.ErrInvalidTranslation),
		errors.Is(err, domain.ErrProfileIDRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
