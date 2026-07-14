package user

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
	"github.com/hanbin/hanbin-back/internal/middleware"
	dramasvc "github.com/hanbin/hanbin-back/internal/service/drama"
	svc "github.com/hanbin/hanbin-back/internal/service/user"
)

// Handler обрабатывает HTTP-запросы, связанные с профилем пользователя.
type Handler struct {
	service      *svc.Service
	dramaService *dramasvc.Service
}

func NewHandler(service *svc.Service, dramaService *dramasvc.Service) *Handler {
	return &Handler{service: service, dramaService: dramaService}
}

// RegisterRoutes регистрирует маршруты.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/profiles", h.handleCollection)
	mux.HandleFunc("/api/v1/profiles/", h.handleItem)
	mux.Handle("/api/v1/users/me", middleware.Auth(http.HandlerFunc(h.Me)))
}

func (h *Handler) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.Create(w, r)
	default:
		methodNotAllowed(w)
	}
}

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

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request, id int64) {
	out, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("ERROR GetByID id=%d: %v", id, err)
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body svc.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	out, err := h.service.Create(r.Context(), body)
	if err != nil {
		log.Printf("ERROR Create: %v", err)
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request, id int64) {
	var body svc.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	out, err := h.service.Update(r.Context(), id, body)
	if err != nil {
		log.Printf("ERROR Update id=%d: %v", id, err)
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.service.Delete(r.Context(), id); err != nil {
		log.Printf("ERROR Delete id=%d: %v", id, err)
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Me godoc — GET /api/v1/users/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	profileID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	profile, err := h.service.GetByID(r.Context(), profileID)
	if err != nil {
		log.Printf("ERROR Me GetByID id=%d: %v", profileID, err)
		writeServiceError(w, err)
		return
	}

	dramas, err := h.dramaService.GetAllByProfileID(r.Context(), profileID)
	if err != nil {
		log.Printf("ERROR Me GetAllByProfileID id=%d: %v", profileID, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if dramas == nil {
		dramas = []dramasvc.DramaOutput{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":   profile.ID,
		"name":      profile.Name,
		"email":     profile.Email,
		"dramas":    dramas,
		"badges":    []any{},
		"countries": countryBreakdown(dramas),
	})
}

// countryEntry — одна строка разбивки по странам в ответе /users/me.
type countryEntry struct {
	Country string `json:"country"`
	Count   int    `json:"count"`
	Percent int    `json:"percent"`
}

// countryBreakdown считает количество дорам по каждой стране и сортирует по убыванию.
// Возвращает пустой слайс (не nil), если дорам нет — чтобы фронт не подменял честную
// пустоту моковыми данными.
func countryBreakdown(dramas []dramasvc.DramaOutput) []countryEntry {
	result := make([]countryEntry, 0)
	if len(dramas) == 0 {
		return result
	}

	counts := make(map[string]int)
	for _, d := range dramas {
		c := strings.TrimSpace(d.Country)
		if c == "" {
			c = "unknown"
		}
		counts[c]++
	}

	total := len(dramas)
	for country, count := range counts {
		result = append(result, countryEntry{
			Country: country,
			Count:   count,
			Percent: int(math.Round(float64(count) / float64(total) * 100)),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Country < result[j].Country // стабильный порядок при равенстве
	})

	return result
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

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "profile not found")
	case errors.Is(err, domain.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, domain.ErrEmailNotUnique):
		writeError(w, http.StatusConflict, "email is already taken")
	case errors.Is(err, domain.ErrProfileExists):
		writeError(w, http.StatusConflict, "profile already exists for this user")
	case errors.Is(err, domain.ErrNameRequired),
		errors.Is(err, domain.ErrNameTooLong),
		errors.Is(err, domain.ErrUserIDRequired):
		writeError(w, http.StatusBadRequest, unwrapMessage(err))
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}


// isError сравнивает err с target через errors.Is (обёртка для читаемости).
func isError(err, target error) bool {
	return errors.Is(err, target)
}

// unwrapMessage возвращает текст ошибки для отдачи клиенту.
func unwrapMessage(err error) string {
	return err.Error()
}

func parseID(urlPath, prefix string) (int64, error) {
	raw := strings.TrimPrefix(urlPath, prefix)
	raw = strings.TrimSuffix(raw, "/")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
