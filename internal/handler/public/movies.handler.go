// Package public содержит эндпоинты, которые сознательно НЕ требуют JWT — сделаны для
// фичи "поделиться списком": получатель ссылки открывает read-only срез чужого списка
// фильмов без регистрации/логина. Это единственное место в бэке, где отдаём чужие данные
// без Authorization — поэтому здесь особенно строго со списком полей, которые уходят наружу
// (см. publicMovie ниже — сознательно НЕ те же поля, что в movie.MovieOutput).
package public

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	moviesvc "github.com/hanbin/hanbin-back/internal/service/movie"
	usersvc "github.com/hanbin/hanbin-back/internal/service/user"
)

// Handler обрабатывает публичные (без JWT) read-only запросы для шаринга списка фильмов.
type Handler struct {
	movieService   *moviesvc.Service
	profileService *usersvc.Service
}

func NewHandler(movieService *moviesvc.Service, profileService *usersvc.Service) *Handler {
	return &Handler{movieService: movieService, profileService: profileService}
}

// RegisterRoutes регистрирует маршруты:
//
//	GET /api/v1/public/profiles/{id}/movies — публичный список фильмов профиля (без JWT).
//	Используется публичной read-only страницей на фронте (см. /#/u/{id}), на которую ведёт
//	кнопка "Поделиться" — получатель ссылки видит список без регистрации.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/public/profiles/", h.handleProfileMovies)
}

func (h *Handler) handleProfileMovies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Ожидаем ровно "/api/v1/public/profiles/{id}/movies" — никаких других под-путей пока нет.
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/public/profiles/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "movies" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	profileID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || profileID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid profile id")
		return
	}

	profile, err := h.profileService.GetByID(r.Context(), profileID)
	if err != nil {
		// Не палим разницу между "профиль не найден" и любой другой ошибкой чтения —
		// снаружи это должно выглядеть одинаково: 404.
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}

	movies, err := h.movieService.GetAllByProfileID(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	publicMovies := make([]publicMovie, 0, len(movies))
	for _, m := range movies {
		if m.IsArchived {
			continue // архив — это личное "не досмотрела/бросила", в публичный шер не попадает
		}
		publicMovies = append(publicMovies, publicMovie{
			Title:       m.Title,
			Genre:       m.Genre,
			Country:     m.Country,
			Category:    m.Category,
			ReleaseYear: m.ReleaseYear,
			WatchStatus: m.WatchStatus,
		})
	}

	writeJSON(w, http.StatusOK, publicProfileMovies{
		ProfileID: profile.ID,
		Name:      profile.Name,
		Movies:    publicMovies,
	})
}

// publicMovie — специально ОТДЕЛЬНЫЙ тип от movie.MovieOutput: здесь нет id/profile_id/
// is_archived/created_at/updated_at — эти поля не нужны для read-only просмотра и не должны
// уходить наружу без авторизации (даже безобидная на первый взгляд метаинформация вроде
// created_at по факту раскрывает то, когда пользователь начал пользоваться сервисом).
type publicMovie struct {
	Title       string `json:"title"`
	Genre       string `json:"genre"`
	Country     string `json:"country"`
	Category    string `json:"category"`
	ReleaseYear *int   `json:"release_year"`
	WatchStatus string `json:"watch_status"`
}

// publicProfileMovies — тело ответа. ProfileID дублируем из URL — фронту удобнее не парсить его
// самому из адреса страницы. Email профиля сюда сознательно не попадает.
type publicProfileMovies struct {
	ProfileID int64         `json:"profile_id"`
	Name      string        `json:"name"`
	Movies    []publicMovie `json:"movies"`
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
