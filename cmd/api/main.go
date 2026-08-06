package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq"

	authhandler          "github.com/hanbin/hanbin-back/internal/handler/auth"
	dramahandler         "github.com/hanbin/hanbin-back/internal/handler/drama"
	moviehandler         "github.com/hanbin/hanbin-back/internal/handler/movie"
	moviecategoryhandler "github.com/hanbin/hanbin-back/internal/handler/moviecategory"
	scraperhandler       "github.com/hanbin/hanbin-back/internal/handler/scraper"
	streamingsitehandler "github.com/hanbin/hanbin-back/internal/handler/streamingsite"
	userhandler          "github.com/hanbin/hanbin-back/internal/handler/user"
	"github.com/hanbin/hanbin-back/internal/mailer"
	"github.com/hanbin/hanbin-back/internal/middleware"
	authrepo          "github.com/hanbin/hanbin-back/internal/repository/auth"
	dramarepo         "github.com/hanbin/hanbin-back/internal/repository/drama"
	movierepo         "github.com/hanbin/hanbin-back/internal/repository/movie"
	moviecategoryrepo "github.com/hanbin/hanbin-back/internal/repository/moviecategory"
	scrapecacherepo   "github.com/hanbin/hanbin-back/internal/repository/scrapecache"
	streamingsiterepo "github.com/hanbin/hanbin-back/internal/repository/streamingsite"
	userrepo          "github.com/hanbin/hanbin-back/internal/repository/user"
	authsvc          "github.com/hanbin/hanbin-back/internal/service/auth"
	dramasvc         "github.com/hanbin/hanbin-back/internal/service/drama"
	moviesvc         "github.com/hanbin/hanbin-back/internal/service/movie"
	moviecategorysvc "github.com/hanbin/hanbin-back/internal/service/moviecategory"
	scrapersvc       "github.com/hanbin/hanbin-back/internal/service/scraper"
	streamingsitesvc "github.com/hanbin/hanbin-back/internal/service/streamingsite"
	usersvc          "github.com/hanbin/hanbin-back/internal/service/user"
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
	userRepo          := userrepo.NewPostgresRepository(db)
	resetTokenRepo    := authrepo.NewPostgresResetTokenRepository(db)
	dramaRepo         := dramarepo.NewPostgresRepository(db)
	movieRepo         := movierepo.NewPostgresRepository(db)
	movieCategoryRepo := moviecategoryrepo.NewPostgresRepository(db)
	scrapeCacheRepo   := scrapecacherepo.NewPostgresRepository(db)
	streamingSiteRepo := streamingsiterepo.NewPostgresRepository(db)
	userService  := usersvc.NewService(userRepo)
	dramaService := dramasvc.NewService(dramaRepo)
	movieService := moviesvc.NewService(movieRepo)
	movieCategoryService := moviecategorysvc.NewService(movieCategoryRepo)
	authService  := authsvc.NewService(userRepo, resetTokenRepo, mailer.NewFromEnv(), origins)
	scrapeService        := scrapersvc.NewService(scrapeCacheRepo) // гибрид: cache-aside с TTL поверх internal/scraper
	streamingSiteService := streamingsitesvc.NewService(streamingSiteRepo)
	userHandler          := userhandler.NewHandler(userService, dramaService)
	dramaHandler         := dramahandler.NewHandler(dramaService)
	movieHandler         := moviehandler.NewHandler(movieService)
	movieCategoryHandler := moviecategoryhandler.NewHandler(movieCategoryService)
	authHandler          := authhandler.NewHandler(authService)
	scrapeHandler        := scraperhandler.NewHandler(scrapeService)
	streamingSiteHandler := streamingsitehandler.NewHandler(streamingSiteService)

	// ── Routing ───────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)          // POST /api/v1/auth/register, /api/v1/auth/login
	userHandler.RegisterRoutes(mux)          // GET  /api/v1/users/me, /api/v1/profiles/...
	streamingSiteHandler.RegisterRoutes(mux) // GET|POST /api/v1/streaming-sites, PATCH|DELETE /api/v1/streaming-sites/{id}
	movieCategoryHandler.RegisterRoutes(mux) // GET|POST /api/v1/movie-categories, PATCH|DELETE /api/v1/movie-categories/{id}

	// ВАЖНО: scrapeHandler регистрируется ДО dramaHandler.
	// Паттерн "GET /api/v1/dramas/scrape" точнее, чем "/api/v1/dramas/",
	// поэтому mux (Go 1.22+) выбирает его без Auth-middleware.
	scrapeHandler.RegisterRoutes(mux) // GET  /api/v1/dramas/scrape  (публичный, без JWT)
	dramaHandler.RegisterRoutes(mux)  // POST /api/v1/dramas, PATCH /api/v1/dramas/{id}/archive
	movieHandler.RegisterRoutes(mux)  // GET|POST /api/v1/movies

	httpHandler := middleware.CORS(origins)(mux)

	log.Printf("hanbin-back listening on %s", addr)
	log.Println("registered routes:")
	log.Println("  POST /api/v1/auth/register")
	log.Println("  POST /api/v1/auth/login")
	log.Println("  POST /api/v1/auth/set-password")
	log.Println("  POST /api/v1/auth/forgot-password")
	log.Println("  POST /api/v1/auth/reset-password")
	log.Println("  POST /api/v1/profiles")
	log.Println("  GET|PATCH|DELETE /api/v1/profiles/{id}")
	log.Println("  GET /api/v1/users/me")
	log.Println("  GET /api/v1/dramas/stats")
	log.Println("  GET|POST /api/v1/streaming-sites")
	log.Println("  PATCH|DELETE /api/v1/streaming-sites/{id}")
	log.Println("  GET|POST /api/v1/movie-categories")
	log.Println("  PATCH|DELETE /api/v1/movie-categories/{id}")
	log.Println("  GET|POST /api/v1/movies")
	log.Println("  GET /api/v1/movies/stats")
	log.Println("  PATCH /api/v1/movies/{id}")
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
