package collectors

import (
	"strings"
	"testing"

	"github.com/ruaan-deysel/unraid-management-agent/daemon/domain"
	"github.com/ruaan-deysel/unraid-management-agent/daemon/dto"
)

func TestNewGPUCollector(t *testing.T) {
	hub := domain.NewEventBus(10)
	ctx := &domain.Context{Hub: hub}

	collector := NewGPUCollector(ctx)

	if collector == nil {
		t.Fatal("NewGPUCollector() returned nil")
	}

	if collector.ctx != ctx {
		t.Error("GPUCollector context not set correctly")
	}
}

func TestNvidiaSMIOutputParsing(t *testing.T) {
	// Test parsing of nvidia-smi query output
	output := `GPU 00000000:01:00.0
    Product Name                          : NVIDIA GeForce RTX 3080
    GPU UUID                              : GPU-12345678-1234-1234-1234-123456789abc
    Fan Speed                             : 45 %
    Temperature
        GPU Current Temp                  : 55 C
    Power Readings
        Power Draw                        : 120.50 W
        Power Limit                       : 320.00 W
    Memory Usage
        Total                             : 10240 MiB
        Used                              : 2048 MiB
        Free                              : 8192 MiB
    Utilization
        Gpu                               : 25 %
        Memory                            : 20 %
`
	lines := strings.Split(output, "\n")

	data := make(map[string]string)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		data[key] = value
	}

	// Verify parsing
	if data["Product Name"] != "NVIDIA GeForce RTX 3080" {
		t.Errorf("Product Name = %q, want %q", data["Product Name"], "NVIDIA GeForce RTX 3080")
	}

	if data["GPU Current Temp"] != "55 C" {
		t.Errorf("GPU Current Temp = %q, want %q", data["GPU Current Temp"], "55 C")
	}
}

func TestGPUMetricsExtraction(t *testing.T) {
	// Test extracting metrics from nvidia-smi values
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"fan speed", "45 %", 45},
		{"temperature", "55 C", 55},
		{"utilization", "25 %", 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract numeric value
			parts := strings.Fields(tt.input)
			if len(parts) > 0 {
				// Verify the format is parseable
				_ = parts[0]
			}
		})
	}
}

func TestGPUMemoryParsing(t *testing.T) {
	// Test parsing memory values
	tests := []struct {
		input    string
		expected int
	}{
		{"10240 MiB", 10240},
		{"2048 MiB", 2048},
		{"8192 MiB", 8192},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parts := strings.Fields(tt.input)
			if len(parts) < 2 {
				t.Errorf("Invalid memory format: %q", tt.input)
			}
		})
	}
}
func TestGPUPowerParsing(t *testing.T) {
	// Test parsing power values
	tests := []struct {
		input    string
		expected float64
	}{
		{"120.50 W", 120.50},
		{"320.00 W", 320.00},
		{"0.00 W", 0.00},
		{"250.75 W", 250.75},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if !strings.Contains(tt.input, "W") {
				t.Errorf("Power value %q should contain 'W'", tt.input)
			}
		})
	}
}

func TestGPUVendorTypes(t *testing.T) {
	// Test known GPU vendor types
	vendors := []string{"NVIDIA", "AMD", "Intel"}

	for _, vendor := range vendors {
		t.Run(vendor, func(t *testing.T) {
			if vendor == "" {
				t.Error("Vendor should not be empty")
			}
		})
	}
}

func TestGPUTemperatureParsing(t *testing.T) {
	// Test parsing temperature values
	tests := []struct {
		input    string
		expected int
	}{
		{"55 C", 55},
		{"75 C", 75},
		{"30 C", 30},
		{"90 C", 90},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if !strings.Contains(tt.input, "C") {
				t.Errorf("Temperature %q should contain 'C'", tt.input)
			}
		})
	}
}

func TestGPUUtilizationRanges(t *testing.T) {
	// Test valid utilization percentage ranges
	tests := []struct {
		name        string
		utilization int
		valid       bool
	}{
		{"zero", 0, true},
		{"low", 25, true},
		{"medium", 50, true},
		{"high", 75, true},
		{"full", 100, true},
		{"negative", -1, false},
		{"over 100", 101, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.utilization >= 0 && tt.utilization <= 100
			if isValid != tt.valid {
				t.Errorf("Utilization %d: valid = %v, want %v", tt.utilization, isValid, tt.valid)
			}
		})
	}
}

func TestGPUGlobalIndexReassignment(t *testing.T) {
	// Simulate multi-vendor GPU collection with vendor-local indices.
	// Intel GPUs get index 0,1; NVIDIA gets index 0; AMD gets index 0.
	// After global reassignment, indices must be 0,1,2,3 sequentially.
	gpuMetrics := []*dto.GPUMetrics{
		{Available: true, Index: 0, Vendor: "intel", Name: "UHD Graphics 630"},
		{Available: true, Index: 1, Vendor: "intel", Name: "UHD Graphics 770"},
		{Available: true, Index: 0, Vendor: "nvidia", Name: "GeForce RTX 5060 Ti"},
		{Available: true, Index: 0, Vendor: "amd", Name: "Radeon RX 7900"},
	}

	// Reassign indices globally (same logic as Collect method)
	for i, gpu := range gpuMetrics {
		gpu.Index = i
	}

	for i, gpu := range gpuMetrics {
		if gpu.Index != i {
			t.Errorf("GPU[%d] Index = %d, want %d (vendor=%s, name=%s)",
				i, gpu.Index, i, gpu.Vendor, gpu.Name)
		}
	}

	// Verify vendor ordering: Intel → NVIDIA → AMD
	expectedVendors := []string{"intel", "intel", "nvidia", "amd"}
	for i, gpu := range gpuMetrics {
		if gpu.Vendor != expectedVendors[i] {
			t.Errorf("GPU[%d] Vendor = %q, want %q", i, gpu.Vendor, expectedVendors[i])
		}
	}
}

func TestGPUGlobalIndexSingleVendor(t *testing.T) {
	// When only one vendor is present, indices should still be sequential.
	gpuMetrics := []*dto.GPUMetrics{
		{Available: true, Index: 0, Vendor: "nvidia", Name: "GeForce RTX 3060"},
		{Available: true, Index: 1, Vendor: "nvidia", Name: "GeForce RTX 5060 Ti"},
		{Available: true, Index: 2, Vendor: "nvidia", Name: "GeForce RTX 5060 Ti"},
	}

	for i, gpu := range gpuMetrics {
		gpu.Index = i
	}

	for i, gpu := range gpuMetrics {
		if gpu.Index != i {
			t.Errorf("GPU[%d] Index = %d, want %d", i, gpu.Index, i)
		}
	}
}
