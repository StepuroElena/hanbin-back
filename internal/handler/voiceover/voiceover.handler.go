package voiceover

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	domain "github.com/hanbin/hanbin-back/internal/domain/voiceover"
	"github.com/hanbin/hanbin-back/internal/middleware"
	svc "github.com/hanbin/hanbin-back/internal/service/voiceover"
)

// Handler обрабатывает HTTP-запросы для озвучек.
type Handler struct {
	service *svc.Service
}

func NewHandler(service *svc.Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes регистрирует маршруты:
//
//	GET    /api/v1/voiceovers           — список озвучек профиля: дефолтные + свои, с тоглами (JWT)
//	POST   /api/v1/voiceovers           — добавить свою озвучку (JWT)
//	PATCH  /api/v1/voiceovers/{id}      — переименовать свою / переключить тогл любой (JWT)
//	DELETE /api/v1/voiceovers/{id}      — удалить свою (JWT)
//	GET    /api/v1/voiceovers/match?q=  — сопоставить строку со скрейпа со списком профиля (JWT опционален: гость — с дефолтными)
//	GET    /api/v1/voiceovers/defaults  — дефолтный набор для гостя (публичный)
//
// Паттерны с методом и точным путём (Go 1.22+) точнее, чем "/api/v1/voiceovers/",
// поэтому /match и /defaults не попадают в handleItem.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/voiceovers", middleware.Auth(http.HandlerFunc(h.handleCollection)))
	mux.Handle("/api/v1/voiceovers/", middleware.Auth(http.HandlerFunc(h.handleItem)))
	mux.Handle("GET /api/v1/voiceovers/match", middleware.OptionalAuth(http.HandlerFunc(h.Match)))
	mux.HandleFunc("GET /api/v1/voiceovers/defaults", h.ListDefaults)
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
	raw := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/voiceovers/"), "/")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid voiceover id")
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

// List — GET /api/v1/voiceovers → 200 []Output
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	out, err := h.service.List(r.Context(), profileID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ListDefaults — GET /api/v1/voiceovers/defaults (без JWT) → 200 []Output
func (h *Handler) ListDefaults(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.ListDefaults(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Create — POST /api/v1/voiceovers, Body: {"name": "..."} → 201 Output | 400 | 409
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

// Update — PATCH /api/v1/voiceovers/{id}, Body: {"name"?: "...", "enabled"?: bool} → 200 Output | 400 | 403 | 404 | 409
func (h *Handler) Update(w http.ResponseWriter, r *http.Request, id int64) {
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
	out, err := h.service.Update(r.Context(), profileID, id, body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Delete — DELETE /api/v1/voiceovers/{id} → 204 | 403 | 404
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, id int64) {
	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.service.Delete(r.Context(), profileID, id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Match — GET /api/v1/voiceovers/match?q=DubLikTV,%20Light%20Breeze → 200 {"name": "DubLik.TV"} ("" — нет совпадения).
// С токеном — по включённым озвучкам профиля, без токена (гость) — по дефолтным.
func (h *Handler) Match(w http.ResponseWriter, r *http.Request) {
	profileID, _ := middleware.UserIDFromContext(r.Context()) // 0 = гость
	out, err := h.service.Match(r.Context(), profileID, r.URL.Query().Get("q"))
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
		writeError(w, http.StatusNotFound, domain.ErrNotFound.Error())
	case errors.Is(err, domain.ErrDuplicateName):
		writeError(w, http.StatusConflict, domain.ErrDuplicateName.Error())
	case errors.Is(err, domain.ErrDefaultReadOnly):
		writeError(w, http.StatusForbidden, domain.ErrDefaultReadOnly.Error())
	case errors.Is(err, domain.ErrNameRequired),
		errors.Is(err, domain.ErrNameTooLong),
		errors.Is(err, domain.ErrProfileIDRequired):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
