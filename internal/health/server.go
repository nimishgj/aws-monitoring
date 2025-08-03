package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"aws-monitoring/pkg/logger"
)

// Server provides HTTP endpoints for health checks
type Server struct {
	manager *Manager
	logger  *logger.Logger
	server  *http.Server
	port    int
}

// NewServer creates a new health check HTTP server
func NewServer(manager *Manager, port int, log *logger.Logger) *Server {
	return &Server{
		manager: manager,
		logger:  log.WithComponent("health-server"),
		port:    port,
	}
}

// Start starts the health check HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	
	// Register health check endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/health/live", s.handleLiveness)
	mux.HandleFunc("/health/ready", s.handleReadiness)
	mux.HandleFunc("/health/detailed", s.handleDetailedHealth)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting health check server", logger.Int("port", s.port))

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Health check server failed", logger.String("error", err.Error()))
		}
	}()

	return nil
}

// Stop gracefully stops the health check HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.logger.Info("Stopping health check server")
	return s.server.Shutdown(ctx)
}

// handleHealth provides a basic health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := s.manager.GetHealth()
	
	// Set status code based on health
	statusCode := s.statusToHTTPCode(health.Status)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"status":    health.Status,
		"timestamp": health.Timestamp,
		"uptime":    health.Uptime.String(),
		"service":   health.ServiceName,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode health response", logger.String("error", err.Error()))
	}
}

// handleLiveness provides a liveness probe endpoint
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Liveness check - just verify the application is running
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode liveness response", logger.String("error", err.Error()))
	}
}

// handleReadiness provides a readiness probe endpoint
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := s.manager.GetHealth()
	
	// Readiness check - verify the application can serve traffic
	// We consider the app ready if it's not unhealthy
	ready := health.Status != StatusUnhealthy
	
	statusCode := http.StatusOK
	if !ready {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"status":    map[bool]string{true: "ready", false: "not_ready"}[ready],
		"timestamp": time.Now(),
		"health":    health.Status,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to encode readiness response", logger.String("error", err.Error()))
	}
}

// handleDetailedHealth provides detailed health check information
func (s *Server) handleDetailedHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := s.manager.GetHealth()
	
	statusCode := s.statusToHTTPCode(health.Status)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(health); err != nil {
		s.logger.Error("Failed to encode detailed health response", logger.String("error", err.Error()))
	}
}

// statusToHTTPCode converts health status to appropriate HTTP status code
func (s *Server) statusToHTTPCode(status Status) int {
	switch status {
	case StatusHealthy:
		return http.StatusOK
	case StatusDegraded:
		return http.StatusOK // Still serving traffic but with warnings
	case StatusUnhealthy:
		return http.StatusServiceUnavailable
	case StatusUnknown:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}