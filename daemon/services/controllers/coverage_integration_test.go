//go:build integration

package controllers

import (
	"context"
	"testing"
)

// ===== Docker Controller - SDK-based methods (requires live Docker daemon) =====

func TestDockerControllerCheckContainerUpdate(t *testing.T) {
	dc := NewDockerController()
	defer dc.Close() //nolint:errcheck
	result, err := dc.CheckContainerUpdate(context.Background(), "nonexistent-container-id")
	if err != nil {
		t.Logf("CheckContainerUpdate error: %v", err)
	}
	_ = result
}

func TestDockerControllerCheckAllContainerUpdates(t *testing.T) {
	dc := NewDockerController()
	defer dc.Close() //nolint:errcheck
	result, err := dc.CheckAllContainerUpdates(context.Background())
	if err != nil {
		t.Logf("CheckAllContainerUpdates error: %v", err)
	}
	_ = result
}

func TestDockerControllerGetContainerSize(t *testing.T) {
	dc := NewDockerController()
	defer dc.Close() //nolint:errcheck
	result, err := dc.GetContainerSize("nonexistent-container-id")
	if err != nil {
		t.Logf("GetContainerSize error: %v", err)
	}
	_ = result
}

func TestDockerControllerUpdateContainer(t *testing.T) {
	dc := NewDockerController()
	defer dc.Close() //nolint:errcheck
	result, err := dc.UpdateContainer("nonexistent-container-id", false)
	if err != nil {
		t.Logf("UpdateContainer error: %v", err)
	}
	_ = result
}

func TestDockerControllerUpdateAllContainers(t *testing.T) {
	dc := NewDockerController()
	defer dc.Close() //nolint:errcheck
	result, err := dc.UpdateAllContainers()
	if err != nil {
		// This is expected if no containers need updating or Docker is not available
		t.Logf("UpdateAllContainers error: %v", err)
	}
	_ = result
}

// ===== VM Controller - Snapshot and Clone (requires live libvirt) =====

func TestVMControllerCreateSnapshot(t *testing.T) {
	ctrl := NewVMController()
	err := ctrl.CreateSnapshot("nonexistent-vm", "test-snap", "test description")
	if err == nil {
		t.Log("CreateSnapshot succeeded (unexpected on macOS)")
	}
}

func TestVMControllerListSnapshots(t *testing.T) {
	ctrl := NewVMController()
	result, err := ctrl.ListSnapshots("nonexistent-vm")
	if err != nil {
		t.Logf("ListSnapshots error (expected): %v", err)
	}
	_ = result
}

func TestVMControllerDeleteSnapshot(t *testing.T) {
	ctrl := NewVMController()
	err := ctrl.DeleteSnapshot("nonexistent-vm", "test-snap")
	if err == nil {
		t.Log("DeleteSnapshot succeeded (unexpected on macOS)")
	}
}

func TestVMControllerCloneVM(t *testing.T) {
	ctrl := NewVMController()
	err := ctrl.CloneVM("nonexistent-vm", "clone-vm")
	if err == nil {
		t.Log("CloneVM succeeded (unexpected on macOS)")
	}
}
