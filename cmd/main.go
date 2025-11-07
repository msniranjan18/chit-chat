package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/msniranjan18/chit-chat/config"
	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/hub"
	"github.com/msniranjan18/chit-chat/pkg/routes"
	"github.com/msniranjan18/chit-chat/pkg/store"

	_ "github.com/msniranjan18/chit-chat/docs"
)

func main() {
	// Set log output to stdout/stderr for Docker
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg := config.Load() // Use config

	log.Printf("Starting ChitChat server on port %s\n", cfg.Server.Port)
	log.Printf("Environment: %s\n", cfg.Server.Env)

	// 1. Initialize Storage
	log.Println("Initializing storage...")
	storage, err := store.NewStore(ctx, cfg.Database.URL, cfg.Redis.URL)
	if err != nil {
		log.Fatalf("Failed to connect to storage: %v", err)
	}
	defer storage.Close()

	// Initialize database schema
	log.Println("Initializing database schema...")
	if err := storage.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Start cleanup worker
	go storage.StartCleanupWorker(1*time.Hour, 24*time.Hour*30)

	// 2. Initialize JWT authentication
	log.Println("Initializing authentication...")
	auth.InitJWT(cfg.JWT.Secret)

	// 3. Initialize WebSocket Hub
	log.Println("Initializing WebSocket hub...")
	wsHub := hub.NewHub(storage)
	go wsHub.Run()
	go wsHub.ListenToRedis()

	// 4. Initialize HTTP router
	log.Println("Setting up routes...")
	router := routes.NewRouter(wsHub, storage)

	// 5. Start HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	log.Printf("ChitChat server starting on http://localhost:%s", cfg.Server.Port)
	log.Println("Server is ready to accept connections")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
