package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ruaandeysel/unraid-management-agent/daemon/dto"
	"github.com/ruaandeysel/unraid-management-agent/daemon/logger"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual system collection
	info := dto.SystemInfo{
		Hostname:  "unraid-server",
		Version:   s.ctx.Version,
		Uptime:    12345,
		CPUUsage:  45.5,
		RAMUsage:  62.3,
		RAMTotal:  32 * 1024 * 1024 * 1024,
		RAMUsed:   20 * 1024 * 1024 * 1024,
		Timestamp: time.Now(),
	}
	respondJSON(w, http.StatusOK, info)
}

func (s *Server) handleArray(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement array status collection
	status := dto.ArrayStatus{
		State:       "started",
		UsedPercent: 75.5,
		NumDisks:    10,
		Timestamp:   time.Now(),
	}
	respondJSON(w, http.StatusOK, status)
}

func (s *Server) handleDisks(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement disk collection
	disks := []dto.DiskInfo{}
	respondJSON(w, http.StatusOK, disks)
}

func (s *Server) handleDisk(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	diskID := vars["id"]
	logger.Debug("Getting disk info for %s", diskID)
	// TODO: Implement single disk lookup
	respondJSON(w, http.StatusNotFound, map[string]string{"error": "not implemented"})
}

func (s *Server) handleShares(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement share collection
	shares := []dto.ShareInfo{}
	respondJSON(w, http.StatusOK, shares)
}

func (s *Server) handleDockerList(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement Docker collection
	containers := []dto.ContainerInfo{}
	respondJSON(w, http.StatusOK, containers)
}

func (s *Server) handleDockerInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	logger.Debug("Getting container info for %s", containerID)
	// TODO: Implement single container lookup
	respondJSON(w, http.StatusNotFound, map[string]string{"error": "not implemented"})
}

func (s *Server) handleVMList(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement VM collection
	vms := []dto.VMInfo{}
	respondJSON(w, http.StatusOK, vms)
}

func (s *Server) handleVMInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["id"]
	logger.Debug("Getting VM info for %s", vmID)
	// TODO: Implement single VM lookup
	respondJSON(w, http.StatusNotFound, map[string]string{"error": "not implemented"})
}

func (s *Server) handleUPS(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement UPS collection
	ups := dto.UPSStatus{
		Connected: false,
		Timestamp: time.Now(),
	}
	respondJSON(w, http.StatusOK, ups)
}

func (s *Server) handleGPU(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement GPU collection
	gpu := dto.GPUMetrics{
		Available: false,
		Timestamp: time.Now(),
	}
	respondJSON(w, http.StatusOK, gpu)
}

// Docker control handlers
func (s *Server) handleDockerStart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	// TODO: Implement Docker control
	logger.Info("Starting container %s", containerID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "Container started",
		Timestamp: time.Now(),
	})
}

func (s *Server) handleDockerStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	logger.Info("Stopping container %s", containerID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "Container stopped",
		Timestamp: time.Now(),
	})
}

func (s *Server) handleDockerRestart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	logger.Info("Restarting container %s", containerID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "Container restarted",
		Timestamp: time.Now(),
	})
}

func (s *Server) handleDockerPause(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	logger.Info("Pausing container %s", containerID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "Container paused",
		Timestamp: time.Now(),
	})
}

func (s *Server) handleDockerUnpause(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	logger.Info("Unpausing container %s", containerID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "Container unpaused",
		Timestamp: time.Now(),
	})
}

// VM control handlers
func (s *Server) handleVMStart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["id"]
	logger.Info("Starting VM %s", vmID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "VM started",
		Timestamp: time.Now(),
	})
}

func (s *Server) handleVMStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["id"]
	logger.Info("Stopping VM %s", vmID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "VM stopped",
		Timestamp: time.Now(),
	})
}

func (s *Server) handleVMRestart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["id"]
	logger.Info("Restarting VM %s", vmID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "VM restarted",
		Timestamp: time.Now(),
	})
}

func (s *Server) handleVMPause(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["id"]
	logger.Info("Pausing VM %s", vmID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "VM paused",
		Timestamp: time.Now(),
	})
}

func (s *Server) handleVMResume(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["id"]
	logger.Info("Resuming VM %s", vmID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "VM resumed",
		Timestamp: time.Now(),
	})
}

func (s *Server) handleVMHibernate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["id"]
	logger.Info("Hibernating VM %s", vmID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "VM hibernated",
		Timestamp: time.Now(),
	})
}

func (s *Server) handleVMForceStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["id"]
	logger.Info("Force stopping VM %s", vmID)
	respondJSON(w, http.StatusOK, dto.Response{
		Success:   true,
		Message:   "VM force stopped",
		Timestamp: time.Now(),
	})
}

// Helper function to respond with JSON
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		logger.Error("Failed to encode JSON response: %v", err)
	}
}
