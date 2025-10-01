#!/bin/bash
# Generate service layer files

PROJECT_DIR="/Users/ruaandeysel/Github/unraid-management-agent"
cd "$PROJECT_DIR"

echo "Generating service layer files..."

# Orchestrator
cat > daemon/services/orchestrator.go << 'EOF'
package services

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/common"
	"github.com/ruaandeysel/unraid-management-agent/daemon/domain"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
	"github.com/ruaandeysel/unraid-management-agent/daemon/services/api"
	"github.com/ruaandeysel/unraid-management-agent/daemon/services/collectors"
)

type Orchestrator struct {
	ctx *domain.Context
}

func CreateOrchestrator(ctx *domain.Context) *Orchestrator {
	return &Orchestrator{ctx: ctx}
}

func (o *Orchestrator) Run() error {
	logger.Info("Starting Unraid Management Agent v%s", o.ctx.Version)

	// Initialize collectors
	systemCollector := collectors.NewSystemCollector(o.ctx)
	arrayCollector := collectors.NewArrayCollector(o.ctx)
	diskCollector := collectors.NewDiskCollector(o.ctx)
	dockerCollector := collectors.NewDockerCollector(o.ctx)
	vmCollector := collectors.NewVMCollector(o.ctx)
	upsCollector := collectors.NewUPSCollector(o.ctx)
	gpuCollector := collectors.NewGPUCollector(o.ctx)
	shareCollector := collectors.NewShareCollector(o.ctx)

	// Start collectors
	go systemCollector.Start(time.Duration(common.IntervalSystem) * time.Second)
	go arrayCollector.Start(time.Duration(common.IntervalArray) * time.Second)
	go diskCollector.Start(time.Duration(common.IntervalDisk) * time.Second)
	go dockerCollector.Start(time.Duration(common.IntervalDocker) * time.Second)
	go vmCollector.Start(time.Duration(common.IntervalVM) * time.Second)
	go upsCollector.Start(time.Duration(common.IntervalUPS) * time.Second)
	go gpuCollector.Start(time.Duration(common.IntervalGPU) * time.Second)
	go shareCollector.Start(time.Duration(common.IntervalShares) * time.Second)

	logger.Success("All collectors started")

	// Start API server
	apiServer := api.NewServer(o.ctx)
	go func() {
		if err := apiServer.Start(); err != nil {
			logger.Error("API server error: %v", err)
		}
	}()

	logger.Success("API server started on port %d", o.ctx.Port)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	sig := <-sigChan

	logger.Warning("Received %s signal, shutting down...", sig)

	// Graceful shutdown
	apiServer.Stop()
	logger.Info("Shutdown complete")

	return nil
}
EOF

# API Server
cat > daemon/services/api/server.go << 'EOF'
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
EOF

# Middleware
cat > daemon/services/api/middleware.go << 'EOF'
package api

import (
	"net/http"
	"time"

	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger.Debug("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		logger.Debug("Completed in %v", time.Since(start))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
EOF

echo "Service layer files generated successfully!"
echo ""
echo "Generated files:"
echo "  - daemon/services/orchestrator.go"
echo "  - daemon/services/api/server.go"
echo "  - daemon/services/api/middleware.go"
echo ""
echo "Next steps:"
echo "  1. Generate handlers (generate_handlers.sh)"
echo "  2. Generate WebSocket implementation (generate_websocket.sh)"
echo "  3. Generate collectors (generate_collectors.sh)"
echo "  4. Generate controllers (generate_controllers.sh)"
