package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sentinel/backend/pkg/api"
	"github.com/sentinel/backend/pkg/config"
	"github.com/sentinel/backend/pkg/storage"
)

var (
	configFile = flag.String("config", "config.yaml", "Path to configuration file")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	log.Printf("Sentinel Server v%s starting...", version)

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize storage
	store := storage.NewMemoryStore()

	// Initialize API server
	apiServer := api.NewServer(cfg, store)

	// Setup router
	router := mux.NewRouter()

	// Agent endpoints
	router.HandleFunc("/api/v1/events", apiServer.HandleAgentWebSocket).Methods("GET")

	// Frontend API endpoints
	router.HandleFunc("/api/v1/hosts", apiServer.GetHosts).Methods("GET")
	router.HandleFunc("/api/v1/hosts/{hostname}/events", apiServer.GetHostEvents).Methods("GET")
	router.HandleFunc("/api/v1/events/stream", apiServer.StreamEvents).Methods("GET")
	router.HandleFunc("/api/v1/stats", apiServer.GetStats).Methods("GET")
	router.HandleFunc("/api/v1/network/connections", apiServer.GetActiveConnections).Methods("GET")
	router.HandleFunc("/api/v1/processes", apiServer.GetActiveProcesses).Methods("GET")

	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// Start HTTP server
	addr := cfg.Server.Listen
	log.Printf("Starting server on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Printf("Sentinel server started successfully")
	log.Printf("Agent endpoint: ws://%s/api/v1/events", addr)
	log.Printf("API endpoint: http://%s/api/v1", addr)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	server.Close()
}
