package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

type Server struct {
	ctx        *domain.Context
	httpServer *http.Server
	router     *mux.Router
	wsHub      *WSHub
}

func NewServer(ctx *domain.Context) *Server {
	s := &Server{
		ctx:    ctx,
		router: mux.NewRouter(),
		wsHub:  NewWSHub(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Apply middleware
	s.router.Use(corsMiddleware)
	s.router.Use(loggingMiddleware)
	s.router.Use(recoveryMiddleware)

	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Health check
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Monitoring endpoints
	api.HandleFunc("/system", s.handleSystem).Methods("GET")
	api.HandleFunc("/array", s.handleArray).Methods("GET")
	api.HandleFunc("/disks", s.handleDisks).Methods("GET")
	api.HandleFunc("/disks/{id}", s.handleDisk).Methods("GET")
	api.HandleFunc("/shares", s.handleShares).Methods("GET")
	api.HandleFunc("/docker", s.handleDockerList).Methods("GET")
	api.HandleFunc("/docker/{id}", s.handleDockerInfo).Methods("GET")
	api.HandleFunc("/vm", s.handleVMList).Methods("GET")
	api.HandleFunc("/vm/{id}", s.handleVMInfo).Methods("GET")
	api.HandleFunc("/ups", s.handleUPS).Methods("GET")
	api.HandleFunc("/gpu", s.handleGPU).Methods("GET")

	// Control endpoints
	api.HandleFunc("/docker/{id}/start", s.handleDockerStart).Methods("POST")
	api.HandleFunc("/docker/{id}/stop", s.handleDockerStop).Methods("POST")
	api.HandleFunc("/docker/{id}/restart", s.handleDockerRestart).Methods("POST")
	api.HandleFunc("/docker/{id}/pause", s.handleDockerPause).Methods("POST")
	api.HandleFunc("/docker/{id}/unpause", s.handleDockerUnpause).Methods("POST")

	api.HandleFunc("/vm/{id}/start", s.handleVMStart).Methods("POST")
	api.HandleFunc("/vm/{id}/stop", s.handleVMStop).Methods("POST")
	api.HandleFunc("/vm/{id}/restart", s.handleVMRestart).Methods("POST")
	api.HandleFunc("/vm/{id}/pause", s.handleVMPause).Methods("POST")
	api.HandleFunc("/vm/{id}/resume", s.handleVMResume).Methods("POST")
	api.HandleFunc("/vm/{id}/hibernate", s.handleVMHibernate).Methods("POST")
	api.HandleFunc("/vm/{id}/force-stop", s.handleVMForceStop).Methods("POST")

	// WebSocket endpoint
	api.HandleFunc("/ws", s.handleWebSocket)
}

func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	// Subscribe to events and broadcast to WebSocket clients
	go s.broadcastEvents()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.ctx.Port),
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Info("HTTP server listening on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error: %v", err)
	}
}

func (s *Server) broadcastEvents() {
	// Subscribe to all event topics
	ch := s.ctx.Hub.Sub("system", "array", "disk", "docker", "vm", "ups", "gpu", "share")

	for msg := range ch {
		s.wsHub.Broadcast(msg)
	}
}
