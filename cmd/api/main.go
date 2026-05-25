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

	"task-manager/internal/config"
	"task-manager/internal/handler"
	"task-manager/internal/handler/middleware"
	"task-manager/internal/repository"
	"task-manager/internal/service"
	"task-manager/internal/worker"
	"task-manager/pkg/logger"

	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := logger.New("info")
	logger.Info("Starting Task Manager API")

	logger.Info("Config values",
		"host", cfg.Database.Host,
		"port", cfg.Database.Port,
		"user", cfg.Database.User,
		"dbname", cfg.Database.DBName)

	dbConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	logger.Info("Connection string", "conn", dbConnStr)

	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", "error", err)
	}

	logger.Info("Database connected successfully")

	userRepo := repository.NewUserRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	notificationService := service.NewNotificationService(3, logger)
	defer notificationService.Shutdown()

	workerPool := worker.NewTaskWorkerPool(5, logger)
	defer workerPool.Shutdown()

	batchProcessor := worker.NewBatchProcessor(taskRepo, 100, 1000, 5*time.Second, logger)
	defer batchProcessor.Shutdown()

	deadlineChecker := worker.NewDeadlineChecker(taskRepo, notificationService, 1*time.Minute, logger)
	deadlineChecker.Start()
	defer deadlineChecker.Stop()

	authService := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpirationHours)
	taskService := service.NewTaskService(taskRepo, notificationService, workerPool, batchProcessor, logger)

	authHandler := handler.NewAuthHandler(authService)
	taskHandler := handler.NewTaskHandler(taskService)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/auth/register", authHandler.Register)
	mux.HandleFunc("/api/auth/login", authHandler.Login)

	authMiddleware := middleware.AuthMiddleware(cfg.JWT.Secret)

	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware(http.HandlerFunc(taskHandler.ListTasks)).ServeHTTP(w, r)
		} else if r.Method == http.MethodPost {
			authMiddleware(http.HandlerFunc(taskHandler.CreateTask)).ServeHTTP(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/tasks/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			authMiddleware(http.HandlerFunc(taskHandler.BatchCreateTasks)).ServeHTTP(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/tasks/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			authMiddleware(http.HandlerFunc(taskHandler.ExportTasks)).ServeHTTP(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware(http.HandlerFunc(taskHandler.GetTask)).ServeHTTP(w, r)
		} else if r.Method == http.MethodPut {
			authMiddleware(http.HandlerFunc(taskHandler.UpdateTask)).ServeHTTP(w, r)
		} else if r.Method == http.MethodDelete {
			authMiddleware(http.HandlerFunc(taskHandler.DeleteTask)).ServeHTTP(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		metrics := taskService.GetWorkerMetrics()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
            "total_jobs": %d,
            "completed_jobs": %d,
            "failed_jobs": %d,
            "average_duration_ms": %d
        }`, metrics.TotalJobs, metrics.CompletedJobs, metrics.FailedJobs, metrics.AverageDuration.Milliseconds())
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "healthy", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("Server starting", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced shutdown", "error", err)
	}

	logger.Info("Server stopped")
}
