//go:build linux
// +build linux

package linux

import (
	"testing"
)

func TestGetAllDockerStats(t *testing.T) {
	stats, err := GetAllDockerStats()
	if err != nil {
		// Docker may not be installed or running - this is OK
		t.Logf("GetAllDockerStats() returned error (Docker may not be installed): %v", err)
		return
	}

	if stats == nil {
		t.Fatal("GetAllDockerStats() returned nil")
	}

	if len(stats) == 0 {
		t.Log("No Docker containers running")
		return
	}

	for i, stat := range stats {
		if stat.Name == "" {
			t.Errorf("Docker[%d]: Name is empty", i)
		}

		if stat.ContainerID == "" {
			t.Errorf("Docker[%d]: ContainerID is empty", i)
		}

		if stat.MemoryPercent < 0 || stat.MemoryPercent > 100 {
			t.Errorf("Docker[%d]: MemoryPercent should be 0-100, got %f", i, stat.MemoryPercent)
		}

		t.Logf("Docker[%d]: %s (%s) - CPU: %.2f%%, Memory: %d bytes (%.2f%%)",
			i, stat.Name, stat.ContainerID[:12], stat.CPU, stat.Memory, stat.MemoryPercent)
	}
}

func TestFindDockerPath(t *testing.T) {
	path := findDockerPath()
	if path == "" {
		t.Log("Docker not found in PATH")
	} else {
		t.Logf("Docker found at: %s", path)
	}
}

func TestIsDockerAvailable(t *testing.T) {
	available := isDockerAvailable()
	if available {
		t.Log("Docker daemon is available")
	} else {
		t.Log("Docker daemon is not available")
	}
}

func TestParseFloatPercent(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"50.00%", 50.00},
		{"0.00%", 0.00},
		{"100.00%", 100.00},
		{"25.5%", 25.5},
		{"", 0},
		{"invalid", 0},
	}

	for _, test := range tests {
		result := parseFloatPercent(test.input)
		if result != test.expected {
			t.Errorf("parseFloatPercent(%q) = %f, expected %f", test.input, result, test.expected)
		}
	}
}

func TestParseMemUsage(t *testing.T) {
	tests := []struct {
		input       string
		expectedUse uint64
		expectedLim uint64
	}{
		{"100MiB / 1GiB", 100 * 1024 * 1024, 1024 * 1024 * 1024},
		{"512MiB / 2GiB", 512 * 1024 * 1024, 2 * 1024 * 1024 * 1024},
		{"1GiB / 4GiB", 1024 * 1024 * 1024, 4 * 1024 * 1024 * 1024},
		{"invalid", 0, 0},
	}

	for _, test := range tests {
		use, lim := parseMemUsage(test.input)
		if use != test.expectedUse {
			t.Errorf("parseMemUsage(%q) usage = %d, expected %d", test.input, use, test.expectedUse)
		}
		if lim != test.expectedLim {
			t.Errorf("parseMemUsage(%q) limit = %d, expected %d", test.input, lim, test.expectedLim)
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
	}{
		{"100B", 100},
		{"1KiB", 1024},
		{"1MiB", 1024 * 1024},
		{"1GiB", 1024 * 1024 * 1024},
		{"1TiB", 1024 * 1024 * 1024 * 1024},
		{"100", 100},
	}

	for _, test := range tests {
		result := parseSize(test.input)
		if result != test.expected {
			t.Errorf("parseSize(%q) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func BenchmarkGetAllDockerStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GetAllDockerStats()
	}
}
