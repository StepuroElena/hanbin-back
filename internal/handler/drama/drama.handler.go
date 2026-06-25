package drama

import (
	"encoding/json"
	"errors"
	"net/http"

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
//	POST /api/v1/dramas  — добавить дораму (требует JWT)
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/dramas", middleware.Auth(http.HandlerFunc(h.handleCollection)))
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
