package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
)

type contextKey string

// UserIDKey — ключ, по которому profile_id хранится в контексте запроса.
const UserIDKey contextKey = "user_id"

var errInvalidToken = errors.New("invalid token")

// jwtSecret читается из env JWT_SECRET; для дев-окружения есть дефолт.
func jwtSecret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("hanbin-dev-secret-change-in-prod")
}

// Auth — middleware, которое требует валидный Bearer JWT-токен.
// При успехе кладёт profile_id (int64) в контекст по ключу UserIDKey.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeAuthError(w, "missing or invalid Authorization header")
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		userID, err := ParseJWT(token, jwtSecret())
		if err != nil {
			writeAuthError(w, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext извлекает profile_id из контекста запроса.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(UserIDKey).(int64)
	return v, ok
}

// ── JWT (HS256, без внешних зависимостей) ────────────────────────────────────

type jwtClaims struct {
	Sub int64 `json:"sub"` // profile_id
}

// ParseJWT проверяет подпись токена и возвращает profile_id из поля sub.
func ParseJWT(token string, secret []byte) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, errInvalidToken
	}

	// Проверяем HMAC-SHA256 подпись
	msg := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return 0, errInvalidToken
	}

	// Декодируем payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, errInvalidToken
	}
	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return 0, errInvalidToken
	}
	if claims.Sub <= 0 {
		return 0, errInvalidToken
	}
	return claims.Sub, nil
}

// IssueJWT выпускает HS256-токен с profile_id в поле sub.
// Используется в тестах и будущем auth-эндпоинте.
func IssueJWT(profileID int64) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	claimsJSON, err := json.Marshal(jwtClaims{Sub: profileID})
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	msg := header + "." + payload
	mac := hmac.New(sha256.New, jwtSecret())
	mac.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return msg + "." + sig, nil
}

// ── вспомогательная ──────────────────────────────────────────────────────────

func writeAuthError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
