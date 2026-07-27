package moviecategory

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	domain "github.com/hanbin/hanbin-back/internal/domain/moviecategory"
	"github.com/hanbin/hanbin-back/internal/middleware"
	svc "github.com/hanbin/hanbin-back/internal/service/moviecategory"
)

// Handler обрабатывает HTTP-запросы для категорий фильмов.
type Handler struct {
	service *svc.Service
}

func NewHandler(service *svc.Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes регистрирует маршруты:
//
//	GET    /api/v1/movie-categories      — список категорий профиля (требует JWT), лениво сеет дефолты
//	POST   /api/v1/movie-categories      — добавить свою категорию (требует JWT)
//	PATCH  /api/v1/movie-categories/{id} — обновить категорию (требует JWT)
//	DELETE /api/v1/movie-categories/{id} — удалить категорию (требует JWT)
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/movie-categories", middleware.Auth(http.HandlerFunc(h.handleCollection)))
	mux.Handle("/api/v1/movie-categories/", middleware.Auth(http.HandlerFunc(h.handleItem)))
}

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

func (h *Handler) handleItem(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.URL.Path, "/api/v1/movie-categories/")
	raw = strings.TrimSuffix(raw, "/")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid movie category id")
		return
	}

	switch r.Method {
	case http.MethodPatch:
		h.Update(w, r, id)
	case http.MethodDelete:
		h.Delete(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// List godoc
//
//	GET /api/v1/movie-categories
//	Header: Authorization: Bearer <token>
//	200 OK  → []CategoryOutput (JSON)
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
//	POST /api/v1/movie-categories
//	Header: Authorization: Bearer <token>
//	Body: CreateInput (JSON)
//	201 Created  → CategoryOutput (JSON)
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

// Update godoc
//
//	PATCH /api/v1/movie-categories/{id}
//	Header: Authorization: Bearer <token>
//	Body: UpdateInput (JSON, все поля опциональны)
//	200 OK  → CategoryOutput (JSON)
//	400 Bad Request
//	401 Unauthorized
//	404 Not Found
func (h *Handler) Update(w http.ResponseWriter, r *http.Request, categoryID int64) {
	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body svc.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	out, err := h.service.Update(r.Context(), profileID, categoryID, body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Delete godoc
//
//	DELETE /api/v1/movie-categories/{id}
//	Header: Authorization: Bearer <token>
//	204 No Content
//	401 Unauthorized
//	404 Not Found
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, categoryID int64) {
	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.Delete(r.Context(), profileID, categoryID); err != nil {
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
		writeError(w, http.StatusNotFound, "movie category not found")
	case errors.Is(err, domain.ErrNameRequired),
		errors.Is(err, domain.ErrNameTooLong),
		errors.Is(err, domain.ErrProfileIDRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
