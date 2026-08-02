package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	authdomain "github.com/hanbin/hanbin-back/internal/domain/auth"
	userdomain "github.com/hanbin/hanbin-back/internal/domain/user"
	svc "github.com/hanbin/hanbin-back/internal/service/auth"
)

// Handler обрабатывает запросы авторизации.
type Handler struct {
	service *svc.Service
}

func NewHandler(service *svc.Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes регистрирует маршруты:
//
//	POST /api/v1/auth/register
//	POST /api/v1/auth/login
//	POST /api/v1/auth/set-password
//	POST /api/v1/auth/forgot-password
//	POST /api/v1/auth/reset-password
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/auth/register", h.handleRegister)
	mux.HandleFunc("/api/v1/auth/login", h.handleLogin)
	mux.HandleFunc("/api/v1/auth/set-password", h.handleSetPassword)
	mux.HandleFunc("/api/v1/auth/forgot-password", h.handleForgotPassword)
	mux.HandleFunc("/api/v1/auth/reset-password", h.handleResetPassword)
}

// handleRegister — POST /api/v1/auth/register
func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body svc.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	out, err := h.service.Register(r.Context(), body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// handleLogin — POST /api/v1/auth/login
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body svc.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	out, err := h.service.Login(r.Context(), body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handleSetPassword — POST /api/v1/auth/set-password
// Устанавливает новый пароль для существующего пользователя по email.
// Используется для исправления пустого password_hash у старых пользователей.
//
//	Body: { "email": "...", "password": "..." }
//	200 OK → { "ok": true }
func (h *Handler) handleSetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body svc.SetPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.SetPassword(r.Context(), body); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleForgotPassword — POST /api/v1/auth/forgot-password
// Требует email существующего аккаунта — если нет такого пользователя, вернётся 404.
//
//	Body: { "email": "..." }
//	200 OK → { "reset_link": "...", "expires_at": "..." } — ВРЕМЕННО, пока нет email-отправки (см. сервис)
func (h *Handler) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body svc.ForgotPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	out, err := h.service.ForgotPassword(r.Context(), body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handleResetPassword — POST /api/v1/auth/reset-password
// Устанавливает новый пароль по токену из ссылки восстановления.
//
//	Body: { "token": "...", "password": "..." }
//	200 OK → { "ok": true }
//	400 Bad Request — токен невалиден/истёк/уже использован, или пароль слишком короткий
func (h *Handler) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body svc.ResetPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.ResetPassword(r.Context(), body); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
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
	case errors.Is(err, authdomain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, authdomain.ErrPasswordRequired),
		errors.Is(err, authdomain.ErrPasswordTooShort),
		errors.Is(err, authdomain.ErrTokenInvalid),
		errors.Is(err, authdomain.ErrTokenExpired),
		errors.Is(err, userdomain.ErrNameRequired),
		errors.Is(err, userdomain.ErrEmailRequired),
		errors.Is(err, userdomain.ErrEmailInvalid),
		errors.Is(err, userdomain.ErrNameTooLong),
		errors.Is(err, userdomain.ErrEmailTooLong):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, userdomain.ErrEmailNotUnique):
		writeError(w, http.StatusConflict, "email is already taken")
	case errors.Is(err, userdomain.ErrNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
