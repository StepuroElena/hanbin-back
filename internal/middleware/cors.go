package middleware

import "net/http"

// CORS добавляет заголовки Cross-Origin Resource Sharing к каждому ответу,
// чтобы фронтенд на другом порту мог делать запросы к API.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Разрешаем только перечисленные origins
			if IsAllowedOrigin(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Preflight-запрос браузера — отвечаем сразу и не идём дальше
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IsAllowedOrigin проверяет, что origin входит в список разрешённых (или там есть "*").
// Экспортирована, так как используется ещё и в auth-сервисе — чтобы ссылка восстановления
// пароля вела на тот же хост, с которого реально пришёл запрос (localhost в dev, прод-домен в проде),
// а не на жёстко зашитый в .env FRONTEND_URL.
func IsAllowedOrigin(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}
