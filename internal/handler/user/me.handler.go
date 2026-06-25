package user

import (
	"net/http"
	"strings"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
)

// RegisterMeRoutes добавляет маршрут текущего пользователя.
//
//	GET /api/v1/users/me → GetMe
func (h *Handler) RegisterMeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/users/me", h.handleMe)
}

// handleMe — обработчик GET /api/v1/users/me.
//
// Требует заголовок: Authorization: Bearer <jwt>
// Возвращает: MeOutput (200 OK) — данные пользователя, список дорам и бэйджей.
func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	token := extractBearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "authorization token required")
		return
	}

	out, err := h.service.GetMe(r.Context(), token)
	if err != nil {
		if isError(err, domain.ErrInvalidCredentials) || isError(err, domain.ErrUserNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

// extractBearerToken достаёт токен из заголовка Authorization: Bearer <token>.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
