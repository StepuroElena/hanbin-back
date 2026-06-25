package user

import (
	"encoding/json"
	"net/http"

	domain "github.com/hanbin/hanbin-back/internal/domain/user"
	svc "github.com/hanbin/hanbin-back/internal/service/user"
)

// RegisterAuthRoutes добавляет маршруты аутентификации.
//
//	POST /api/v1/auth/register → Register
//	POST /api/v1/auth/login    → Login
func (h *Handler) RegisterAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/auth/register", h.handleRegister)
	mux.HandleFunc("/api/v1/auth/login", h.handleLogin)
}

// handleRegister — обработчик POST /api/v1/auth/register.
//
// Принимает: { "email": "...", "password": "..." }
// Возвращает: { "user_id": 42 }  (201 Created)
func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var body svc.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	out, err := h.service.Register(r.Context(), body)
	if err != nil {
		writeRegisterError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, out)
}

// handleLogin — обработчик POST /api/v1/auth/login.
//
// Принимает: { "email": "...", "password": "..." }
// Возвращает: { "user_id": 42, "email": "user@example.com", "token": "<jwt>" }  (200 OK)
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var body svc.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	out, err := h.service.Login(r.Context(), body)
	if err != nil {
		writeLoginError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

// writeLoginError маппит ошибки логина на HTTP-статусы.
func writeLoginError(w http.ResponseWriter, err error) {
	switch {
	case isError(err, domain.ErrEmailRequired),
		isError(err, domain.ErrPasswordRequired):
		writeError(w, http.StatusBadRequest, unwrapMessage(err))
	case isError(err, domain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid email or password")
	default:
		writeServiceError(w, err)
	}
}

// writeRegisterError расширяет writeServiceError ошибками, специфичными для регистрации.
func writeRegisterError(w http.ResponseWriter, err error) {
	switch {
	case isError(err, domain.ErrPasswordRequired),
		isError(err, domain.ErrPasswordTooShort):
		writeError(w, http.StatusBadRequest, unwrapMessage(err))
	default:
		writeServiceError(w, err)
	}
}
