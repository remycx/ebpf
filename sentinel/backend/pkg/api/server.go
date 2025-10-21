package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/sentinel/backend/pkg/config"
	"github.com/sentinel/backend/pkg/storage"
)

type Server struct {
	config   *config.Config
	store    *storage.Store
	upgrader websocket.Upgrader
}

func NewServer(cfg *config.Config, store *storage.Store) *Server {
	return &Server{
		config: cfg,
		store:  store,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}
}

// HandleAgentWebSocket handles WebSocket connections from monitoring agents
func (s *Server) HandleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	clientAddr := r.RemoteAddr
	log.Printf("Agent connected: %s", clientAddr)

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error from %s: %v", clientAddr, err)
			}
			break
		}

		if messageType == websocket.BinaryMessage || messageType == websocket.TextMessage {
			if err := s.store.AddEvents(data); err != nil {
				log.Printf("Failed to process events from %s: %v", clientAddr, err)
			}
		}
	}

	log.Printf("Agent disconnected: %s", clientAddr)
}

// GetHosts returns list of monitored hosts
func (s *Server) GetHosts(w http.ResponseWriter, r *http.Request) {
	hosts := s.store.GetHosts()
	respondJSON(w, http.StatusOK, hosts)
}

// GetHostEvents returns events for a specific host
func (s *Server) GetHostEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hostname := vars["hostname"]

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	events := s.store.GetHostEvents(hostname, limit)
	respondJSON(w, http.StatusOK, events)
}

// StreamEvents streams real-time events to frontend via WebSocket
func (s *Server) StreamEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("Frontend client connected: %s", r.RemoteAddr)

	// Subscribe to events
	eventChan := s.store.Subscribe()
	defer s.store.Unsubscribe(eventChan)

	// Send events to client
	for event := range eventChan {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Failed to marshal event: %v", err)
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Failed to send event to client: %v", err)
			break
		}
	}

	log.Printf("Frontend client disconnected: %s", r.RemoteAddr)
}

// GetStats returns overall statistics
func (s *Server) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := s.store.GetStats()
	respondJSON(w, http.StatusOK, stats)
}

// GetActiveConnections returns active network connections
func (s *Server) GetActiveConnections(w http.ResponseWriter, r *http.Request) {
	connections := s.store.GetActiveConnections()
	respondJSON(w, http.StatusOK, connections)
}

// GetActiveProcesses returns active processes
func (s *Server) GetActiveProcesses(w http.ResponseWriter, r *http.Request) {
	processes := s.store.GetActiveProcesses()
	respondJSON(w, http.StatusOK, processes)
}

// Helper function to send JSON responses
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// Helper function to send error responses
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
