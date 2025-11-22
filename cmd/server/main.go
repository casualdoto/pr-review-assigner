package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pr-review-assigner/internal/api"
	"pr-review-assigner/internal/config"
	"pr-review-assigner/internal/handler"
	"pr-review-assigner/internal/service"
	"pr-review-assigner/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Подключение к PostgreSQL с retry логикой
	db, err := connectDBWithRetry(cfg, 5, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Connected to database")

	// Инициализация репозиториев
	repo := storage.NewRepository(db)
	teamRepo := storage.NewTeamRepository(repo)
	userRepo := storage.NewUserRepository(repo)
	prRepo := storage.NewPRRepository(repo)

	// Инициализация сервисов
	teamService := service.NewTeamService(teamRepo, userRepo)
	userService := service.NewUserService(userRepo, prRepo, teamRepo)
	prService := service.NewPRService(prRepo, userRepo, teamRepo)

	// Инициализация handlers
	server := handler.NewServer(teamService, userService, prService)

	// Настройка HTTP сервера
	router := chi.NewRouter()

	// CORS middleware для Swagger UI
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	apiHandler := api.Handler(server)
	router.Mount("/", apiHandler)

	// Статическая отдача OpenAPI спецификации для Swagger UI
	router.Get("/openapi.yml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/openapi.yml")
	})

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerPort),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on port %d", cfg.ServerPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Ожидание сигнала для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// connectDBWithRetry подключается к БД с повторными попытками
func connectDBWithRetry(cfg *config.Config, maxRetries int, retryInterval time.Duration) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", cfg.DSN())
		if err != nil {
			log.Printf("Attempt %d/%d: Failed to open database connection: %v", i+1, maxRetries, err)
			time.Sleep(retryInterval)
			continue
		}

		if err = db.Ping(); err != nil {
			log.Printf("Attempt %d/%d: Failed to ping database: %v", i+1, maxRetries, err)
			db.Close()
			time.Sleep(retryInterval)
			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
}
