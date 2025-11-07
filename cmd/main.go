package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/msniranjan18/chit-chat/pkg/auth"
	"github.com/msniranjan18/chit-chat/pkg/hub"
	"github.com/msniranjan18/chit-chat/pkg/routes"
	"github.com/msniranjan18/chit-chat/pkg/store"

	_ "github.com/msniranjan18/chit-chat/docs" // swagger api-docs
)

func main() {
	time.Sleep(10 * time.Second)
	// Initialize context
	ctx := context.Background()

	// Load configuration from environment variables
	pgConn := os.Getenv("DATABASE_URL")
	log.Println("DATABASE_URL", pgConn)
	if pgConn == "" {
		pgConn = "postgres://postgres:password@localhost:5432/chitchat?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_URL")
	log.Println("REDIS_URL", redisAddr)
	if redisAddr == "" {
		redisAddr = "redis://localhost:6379"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	log.Println("JWT_SECRET:", jwtSecret)
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production-for-chitchat-app"
	}

	port := os.Getenv("PORT")
	log.Println("PORT:", port)
	if port == "" {
		port = "8080"
	}

	// 1. Initialize Storage
	log.Println("Initializing storage...")
	storage, err := store.NewStore(ctx, pgConn, redisAddr)
	if err != nil {
		log.Fatalf("Failed to connect to storage: %v", err)
	}
	defer storage.Close()

	// Initialize database schema
	log.Println("Initializing database schema...")
	if err := storage.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Start cleanup worker for old sessions/messages
	go storage.StartCleanupWorker(1*time.Hour, 24*time.Hour*30) // Clean 30-day old data

	// 2. Initialize JWT authentication
	log.Println("Initializing authentication...")
	//auth.InitJWT([]byte(jwtSecret))
	auth.InitJWT(jwtSecret)

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
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("ChitChat server starting on http://localhost:%s", port)
	log.Println("Server is ready to accept connections")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
