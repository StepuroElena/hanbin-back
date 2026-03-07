package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
	svc "github.com/hanbin/hanbin-back/internal/service/user"
)

// Handler обрабатывает HTTP-запросы, связанные с профилем пользователя.
// Знает только об интерфейсе сервиса — не трогает репозиторий и домен напрямую.
type Handler struct {
	service *svc.Service
}

// NewHandler — конструктор с внедрением зависимости.
func NewHandler(service *svc.Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes регистрирует все маршруты хэндлера в переданном ServeMux.
//
//	GET    /api/v1/profiles/{id}  → GetByID
//	POST   /api/v1/profiles       → Create
//	PATCH  /api/v1/profiles/{id}  → Update
//	DELETE /api/v1/profiles/{id}  → Delete
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/profiles", h.handleCollection)
	mux.HandleFunc("/api/v1/profiles/", h.handleItem)
}

// ── Диспетчеры ────────────────────────────────────────────────────────────────

// handleCollection обрабатывает запросы без ID: POST /api/v1/profiles
func (h *Handler) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.Create(w, r)
	default:
		methodNotAllowed(w)
	}
}

// handleItem обрабатывает запросы с ID: /api/v1/profiles/{id}
func (h *Handler) handleItem(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/v1/profiles/")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id: must be a positive integer")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.GetByID(w, r, id)
	case http.MethodPatch:
		h.Update(w, r, id)
	case http.MethodDelete:
		h.Delete(w, r, id)
	default:
		methodNotAllowed(w)
	}
}

// ── Хэндлеры ─────────────────────────────────────────────────────────────────

// GetByID godoc
//
//	GET /api/v1/profiles/{id}
//	200 OK        → ProfileOutput (JSON)
//	400 Bad Request
//	404 Not Found
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request, id int64) {
	out, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Create godoc
//
//	POST /api/v1/profiles
//	Body: {"name": "...", "email": "..."}
//	201 Created   → ProfileOutput (JSON)
//	400 Bad Request
//	409 Conflict  (email already taken)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body svc.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	out, err := h.service.Create(r.Context(), body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// Update godoc
//
//	PATCH /api/v1/profiles/{id}
//	Body: {"name": "...", "email": "..."}  (оба поля опциональны)
//	200 OK        → ProfileOutput (JSON)
//	400 Bad Request
//	404 Not Found
//	409 Conflict
func (h *Handler) Update(w http.ResponseWriter, r *http.Request, id int64) {
	var body svc.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	out, err := h.service.Update(r.Context(), id, body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Delete godoc
//
//	DELETE /api/v1/profiles/{id}
//	204 No Content
//	404 Not Found
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.service.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Вспомогательные функции ───────────────────────────────────────────────────

// errorResponse — структура ошибки в JSON-ответе.
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

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// writeServiceError маппит доменные ошибки на HTTP-статусы.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "profile not found")
	case errors.Is(err, domain.ErrEmailNotUnique):
		writeError(w, http.StatusConflict, "email is already taken")
	case errors.Is(err, domain.ErrNameRequired),
		errors.Is(err, domain.ErrEmailRequired),
		errors.Is(err, domain.ErrNameTooLong),
		errors.Is(err, domain.ErrEmailTooLong),
		errors.Is(err, domain.ErrEmailInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// parseID вырезает числовой ID из пути вида /api/v1/profiles/42
func parseID(urlPath, prefix string) (int64, error) {
	raw := strings.TrimPrefix(urlPath, prefix)
	raw = strings.TrimSuffix(raw, "/")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
