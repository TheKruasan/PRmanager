package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"prmanager/internal/handlers"
	"prmanager/internal/repository"
	"prmanager/internal/service"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := log.New(os.Stdout, "PR-REVIEWER: ", log.LstdFlags|log.Lshortfile)

	dbURL := getEnv("DATABASE_URL", "postgres://postgres:password@postgres:5432/PRmanager?sslmode=disable")
	serverAddr := getEnv("SERVER_ADDR", ":8080")

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		logger.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	logger.Println("Database connection established")

	repo := repository.NewRepository(dbPool, logger)
	svc := service.NewService(repo, logger)
	handler := handlers.NewHandler(svc, logger)

	router := chi.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next.ServeHTTP(w, r)
		})
	})

	router.Post("/team/add", handler.TeamHandler.CreateTeam)
	router.Get("/team/get", handler.TeamHandler.GetTeam)
	router.Post("/users/setIsActive", handler.UserHandler.SetUserActive)
	router.Get("/users/getReview", handler.UserHandler.GetUserReviews)
	router.Post("/pullRequest/create", handler.PullRequestHandler.CreatePullRequest)
	router.Post("/pullRequest/merge", handler.PullRequestHandler.MergePullRequest)
	router.Post("/pullRequest/reassign", handler.PullRequestHandler.ReassignReviewer)

	server := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	go func() {
		logger.Printf("Server starting on %s", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
