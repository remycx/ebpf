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
	"github.com/sentinel/backend/pkg/auth"
	"github.com/sentinel/backend/pkg/config"
	"github.com/sentinel/backend/pkg/middleware"
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

	// Initialize authentication
	authManager := auth.NewAuthManager()

	// Load API keys if authentication is required
	if cfg.Security.RequireAuth {
		keys, err := auth.LoadKeysFromFile(cfg.Security.APIKeysFile)
		if err != nil {
			log.Printf("Warning: failed to load API keys: %v", err)
			log.Println("Creating bootstrap key...")
			bootstrapKey := auth.CreateBootstrapKey()
			keys = []*auth.APIKey{bootstrapKey}
			if err := auth.SaveKeysToFile(cfg.Security.APIKeysFile, keys); err != nil {
				log.Printf("Warning: failed to save bootstrap key: %v", err)
			} else {
				log.Printf("Bootstrap key created and saved to %s", cfg.Security.APIKeysFile)
				log.Println("IMPORTANT: Use key 'sentinel-bootstrap-key-change-me' for initial setup")
			}
		}

		for _, key := range keys {
			authManager.LoadKey(key)
		}
		log.Printf("Loaded %d API key(s)", len(keys))
	} else {
		log.Println("WARNING: Authentication is disabled. Enable it in production!")
	}

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authManager)
	rateLimiter := middleware.NewRateLimiter(cfg.Security.RateLimit, cfg.Security.RateLimitBurst)
	rateLimiter.CleanupVisitors()

	// Initialize API server
	apiServer := api.NewServer(cfg, store)

	// Setup router
	router := mux.NewRouter()

	// Agent endpoints (with authentication if enabled)
	agentHandler := apiServer.HandleAgentWebSocket
	if cfg.Security.RequireAuth {
		agentHandler = authMiddleware.ValidateAgent(agentHandler)
	}
	router.HandleFunc("/api/v1/events", agentHandler).Methods("GET")

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

	// Apply middleware stack
	handler := c.Handler(router)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.LoggingMiddleware(handler)
	handler = rateLimiter.Limit(handler)

	// Start HTTP/HTTPS server
	addr := cfg.Server.Listen
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		if cfg.Server.TLS.Enabled {
			log.Printf("Starting HTTPS server on %s", addr)
			log.Printf("Using TLS cert: %s, key: %s", cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
			if err := server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		} else {
			log.Printf("Starting HTTP server on %s", addr)
			log.Println("WARNING: TLS is disabled. Enable it in production!")
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}
	}()

	protocol := "http"
	wsProtocol := "ws"
	if cfg.Server.TLS.Enabled {
		protocol = "https"
		wsProtocol = "wss"
	}

	log.Printf("Sentinel server started successfully")
	log.Printf("Security: Auth=%v, TLS=%v, RateLimit=%.1f req/s",
		cfg.Security.RequireAuth, cfg.Server.TLS.Enabled, cfg.Security.RateLimit)
	log.Printf("Agent endpoint: %s://%s/api/v1/events", wsProtocol, addr)
	log.Printf("API endpoint: %s://%s/api/v1", protocol, addr)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	server.Close()
}
