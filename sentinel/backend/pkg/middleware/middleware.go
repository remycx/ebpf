package middleware

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sentinel/backend/pkg/auth"
	"golang.org/x/time/rate"
)

// AuthMiddleware validates API keys for agent connections
type AuthMiddleware struct {
	authManager *auth.AuthManager
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authManager *auth.AuthManager) *AuthMiddleware {
	return &AuthMiddleware{
		authManager: authManager,
	}
}

// ValidateAgent middleware for WebSocket connections from agents
func (am *AuthMiddleware) ValidateAgent(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <api-key>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		apiKey := parts[1]

		// Validate key
		key, err := am.authManager.ValidateKey(apiKey)
		if err != nil {
			log.Printf("Authentication failed: %v", err)
			http.Error(w, "Invalid or expired API key", http.StatusUnauthorized)
			return
		}

		log.Printf("Agent authenticated: %s (key: %s)", r.RemoteAddr, key.Name)

		// Call next handler
		next(w, r)
	}
}

// RateLimiter implements rate limiting per IP address
type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
// rate: requests per second
// burst: maximum burst size
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		rate:     rate.Limit(rps),
		burst:    burst,
	}
}

// getVisitor returns the rate limiter for a given IP
func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = limiter
	}

	return limiter
}

// Limit middleware that enforces rate limiting
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get IP from X-Real-IP header or RemoteAddr
		ip := r.Header.Get("X-Real-IP")
		if ip == "" {
			ip = r.RemoteAddr
		}

		limiter := rl.getVisitor(ip)
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			log.Printf("Rate limit exceeded for IP: %s", ip)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// CleanupVisitors removes old visitors to prevent memory leaks
func (rl *RateLimiter) CleanupVisitors() {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			// Simple cleanup: clear all visitors
			// In production, track last access time and only remove stale entries
			rl.visitors = make(map[string]*rate.Limiter)
			rl.mu.Unlock()
		}
	}()
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs all requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
