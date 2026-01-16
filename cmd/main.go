package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/msniranjan18/chit-chat/config"
	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/hub"
	"github.com/msniranjan18/chit-chat/pkg/middleware"
	"github.com/msniranjan18/chit-chat/pkg/routes"
	"github.com/msniranjan18/chit-chat/pkg/store"

	_ "github.com/msniranjan18/chit-chat/docs"
)

func main() {
	// Initialize structured logger
	logger := setupLogger()
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg := config.Load()

	slog.Info("Starting ChitChat server",
		"port", cfg.Server.Port,
		"environment", cfg.Server.Env,
		"log_level", getLogLevel())

	// 1. Initialize Storage
	slog.Info("Initializing storage...")
	storage, err := store.NewStore(ctx, cfg.Database.URL, cfg.Redis.URL, logger)
	if err != nil {
		slog.Error("Failed to connect to storage", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			slog.Error("Failed to close storage", "error", err)
		}
	}()

	// Initialize database schema
	slog.Info("Initializing database schema...")
	if err := storage.InitSchema(); err != nil {
		slog.Error("Failed to initialize schema", "error", err)
		os.Exit(1)
	}

	// Start cleanup worker
	go storage.StartCleanupWorker(1*time.Hour, 24*time.Hour*30)
	slog.Debug("Cleanup worker started", "interval", "1h", "max_age", "30d")

	// 2. Initialize JWT authentication
	slog.Info("Initializing authentication...")
	auth.InitJWT(cfg.JWT.Secret)

	// 3. Initialize WebSocket Hub
	slog.Info("Initializing WebSocket hub...")
	wsHub := hub.NewHub(storage, logger)
	go wsHub.Run()
	go wsHub.ListenToRedis()
	slog.Debug("WebSocket hub initialized and running")

	// 4. Initialize HTTP router
	slog.Info("Setting up routes...")
	router := routes.NewRouter(wsHub, storage, logger)

	// Apply middleware
	handler := middleware.LoggingMiddleware(router, logger)

	// 5. Start HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	slog.Info("ChitChat server starting",
		"address", server.Addr,
		"read_timeout", cfg.Server.ReadTimeout,
		"write_timeout", cfg.Server.WriteTimeout,
		"idle_timeout", cfg.Server.IdleTimeout)

	slog.Info("Server is ready to accept connections")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

func setupLogger() *slog.Logger {
	// Check environment for log level
	logLevel := getLogLevel()

	// Create JSON handler for production, text handler for development
	var handler slog.Handler
	if os.Getenv("ENVIRONMENT") == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Customize attribute names for development
				if a.Key == slog.TimeKey {
					a.Key = "timestamp"
				} else if a.Key == slog.LevelKey {
					a.Key = "level"
				} else if a.Key == slog.MessageKey {
					a.Key = "message"
				}
				return a
			},
		})
	}

	return slog.New(handler)
}

func getLogLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		// Default to info for production, debug for development
		if os.Getenv("ENVIRONMENT") == "production" {
			return slog.LevelInfo
		}
		return slog.LevelDebug
	}
}
