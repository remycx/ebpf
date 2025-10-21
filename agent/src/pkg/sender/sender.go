package sender

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sentinel/agent/pkg/config"
)

type Sender struct {
	config     config.ServerConfig
	conn       *websocket.Conn
	mu         sync.Mutex
	reconnect  chan struct{}
	connected  bool
}

func New(cfg config.ServerConfig) (*Sender, error) {
	s := &Sender{
		config:    cfg,
		reconnect: make(chan struct{}, 1),
	}

	if err := s.connect(); err != nil {
		log.Printf("Initial connection failed: %v. Will retry in background.", err)
		go s.reconnectLoop()
	}

	return s, nil
}

func (s *Sender) connect() error {
	u, err := url.Parse(s.config.URL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	dialer := websocket.DefaultDialer
	if s.config.InsecureSkipTLS {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Add authentication header if API key is provided
	headers := make(http.Header)
	if s.config.APIKey != "" {
		headers.Add("Authorization", "Bearer "+s.config.APIKey)
	}

	conn, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	s.mu.Lock()
	s.conn = conn
	s.connected = true
	s.mu.Unlock()

	log.Printf("Connected to Sentinel server at %s", s.config.URL)

	// Start ping/pong handler
	go s.pingHandler()

	return nil
}

func (s *Sender) reconnectLoop() {
	retries := 0
	for {
		<-s.reconnect

		if s.config.MaxRetries > 0 && retries >= s.config.MaxRetries {
			log.Printf("Max retries (%d) reached. Giving up.", s.config.MaxRetries)
			return
		}

		delay := time.Duration(s.config.ReconnectDelay) * time.Second
		log.Printf("Reconnecting in %v... (attempt %d)", delay, retries+1)
		time.Sleep(delay)

		if err := s.connect(); err != nil {
			log.Printf("Reconnection failed: %v", err)
			retries++
			s.reconnect <- struct{}{}
		} else {
			retries = 0
		}
	}
}

func (s *Sender) pingHandler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		if s.conn != nil {
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping failed: %v", err)
				s.connected = false
				s.conn.Close()
				s.conn = nil
				s.mu.Unlock()
				s.reconnect <- struct{}{}
				return
			}
		}
		s.mu.Unlock()
	}
}

func (s *Sender) Send(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected || s.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	if err := s.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		s.connected = false
		s.conn.Close()
		s.conn = nil
		s.reconnect <- struct{}{}
		return fmt.Errorf("failed to send data: %w", err)
	}

	return nil
}

func (s *Sender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
		s.connected = false
	}

	return nil
}
