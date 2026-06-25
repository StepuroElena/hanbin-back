package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq"

	authhandler    "github.com/hanbin/hanbin-back/internal/handler/auth"
	dramahandler   "github.com/hanbin/hanbin-back/internal/handler/drama"
	scraperhandler "github.com/hanbin/hanbin-back/internal/handler/scraper"
	userhandler    "github.com/hanbin/hanbin-back/internal/handler/user"
	"github.com/hanbin/hanbin-back/internal/middleware"
	dramarepo "github.com/hanbin/hanbin-back/internal/repository/drama"
	userrepo  "github.com/hanbin/hanbin-back/internal/repository/user"
	authsvc  "github.com/hanbin/hanbin-back/internal/service/auth"
	dramasvc "github.com/hanbin/hanbin-back/internal/service/drama"
	usersvc  "github.com/hanbin/hanbin-back/internal/service/user"
)

func main() {
	dsn     := getenv("DATABASE_URL", "host=localhost port=5432 user=elenastepuro dbname=hanbin sslmode=disable")
	addr    := getenv("ADDR", ":8080")
	origins := strings.Split(getenv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5500,http://127.0.0.1:5500"), ",")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	log.Println("connected to database")

// ── Dependency Injection ──────────────────────────────────────────────────
	userRepo  := userrepo.NewPostgresRepository(db)
	dramaRepo := dramarepo.NewPostgresRepository(db)
	userService  := usersvc.NewService(userRepo)
	dramaService := dramasvc.NewService(dramaRepo)
	authService  := authsvc.NewService(userRepo)
	userHandler   := userhandler.NewHandler(userService, dramaService)
	dramaHandler  := dramahandler.NewHandler(dramaService)
	authHandler   := authhandler.NewHandler(authService)
	scrapeHandler := scraperhandler.NewHandler()
	// ── Routing ───────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)   // POST /api/v1/auth/register, /api/v1/auth/login
	userHandler.RegisterRoutes(mux)   // GET  /api/v1/users/me, /api/v1/profiles/...

	// ВАЖНО: scrapeHandler регистрируется ДО dramaHandler.
	// Паттерн "GET /api/v1/dramas/scrape" точнее, чем "/api/v1/dramas/",
	// поэтому mux (Go 1.22+) выбирает его без Auth-middleware.
	scrapeHandler.RegisterRoutes(mux) // GET  /api/v1/dramas/scrape  (публичный, без JWT)
	dramaHandler.RegisterRoutes(mux)  // POST /api/v1/dramas, PATCH /api/v1/dramas/{id}/archive

	httpHandler := middleware.CORS(origins)(mux)

	log.Printf("hanbin-back listening on %s", addr)
	log.Println("registered routes:")
	log.Println("  POST /api/v1/auth/register")
	log.Println("  POST /api/v1/auth/login")
	log.Println("  POST /api/v1/profiles")
	log.Println("  GET|PATCH|DELETE /api/v1/profiles/{id}")
	log.Println("  GET /api/v1/users/me")
	log.Printf("allowed origins: %v", origins)

	if err := http.ListenAndServe(addr, httpHandler); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
