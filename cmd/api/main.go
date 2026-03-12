package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq"

	userhandler "github.com/hanbin/hanbin-back/internal/handler/user"
	"github.com/hanbin/hanbin-back/internal/middleware"
	userrepo "github.com/hanbin/hanbin-back/internal/repository/user"
	usersvc "github.com/hanbin/hanbin-back/internal/service/user"
)

func main() {
	dsn := getenv("DATABASE_URL", "host=localhost port=5432 user=elenastepuro dbname=hanbin sslmode=disable")
	addr := getenv("ADDR", ":8080")

	// Список origins, которым разрешено обращаться к API.
	// Задаётся через ALLOWED_ORIGINS="http://localhost:3000,http://localhost:5500"
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

	// Dependency Injection: repo → service → handler
	repo := userrepo.NewPostgresRepository(db)
	userRepo := userrepo.NewPostgresUserRepository(db)
	dramaRepo := userrepo.NewPostgresDramaRepository(db)
	badgeRepo := userrepo.NewPostgresBadgeRepository(db)
	service := usersvc.NewService(repo, userRepo, dramaRepo, badgeRepo)
	handler := userhandler.NewHandler(service)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	handler.RegisterAuthRoutes(mux)
	handler.RegisterMeRoutes(mux)

	// Оборачиваем mux в CORS-middleware
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
