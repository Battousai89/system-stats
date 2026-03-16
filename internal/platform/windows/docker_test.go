package windows

import (
	"testing"

	"system-stats/internal/types"
)

func TestGetAllDockerStats(t *testing.T) {
	stats, err := GetAllDockerStats()
	if err != nil {
		t.Logf("GetAllDockerStats() returned error (docker may not be installed): %v", err)
		return
	}

	if len(stats) == 0 {
		t.Log("GetAllDockerStats() returned empty slice (no running containers)")
		return
	}

	for i, s := range stats {
		if s.Name == "" {
			t.Errorf("Docker[%d]: Name is empty", i)
		}

		if s.ContainerID == "" {
			t.Errorf("Docker[%d]: ContainerID is empty", i)
		}

		t.Logf("Docker[%d]: %s (%s) - CPU: %.2f%%, Memory: %d",
			i, s.Name, s.ContainerID[:12], s.CPU, s.Memory)
	}
}

func TestDockerStatsToPrint(t *testing.T) {
	stats, err := GetAllDockerStats()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := types.DockerStatsToPrint(stats)
	if output == "" {
		t.Error("DockerStatsToPrint() returned empty string")
	}

	t.Logf("DockerStatsToPrint():\n%s", output)
}

func TestParseFloatPercent(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"50.00%", 50.00},
		{"0.00%", 0.00},
		{"100.00%", 100.00},
		{"0.50%", 0.50},
		{"", 0},
	}

	for _, test := range tests {
		result := parseFloatPercent(test.input)
		if result != test.expected {
			t.Errorf("parseFloatPercent(%q) = %f, expected %f", test.input, result, test.expected)
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
		{"1.5GiB", uint64(1.5 * 1024 * 1024 * 1024)},
		{"100MB", 100 * 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
	}

	for _, test := range tests {
		result := parseSize(test.input)
		if result != test.expected {
			t.Errorf("parseSize(%q) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestParseMemUsage(t *testing.T) {
	usage, limit := parseMemUsage("100MiB / 1GiB")
	
	expectedUsage := uint64(100 * 1024 * 1024)
	expectedLimit := uint64(1024 * 1024 * 1024)
	
	if usage != expectedUsage {
		t.Errorf("parseMemUsage() usage = %d, expected %d", usage, expectedUsage)
	}
	if limit != expectedLimit {
		t.Errorf("parseMemUsage() limit = %d, expected %d", limit, expectedLimit)
	}
}

// Benchmark тесты
func BenchmarkGetAllDockerStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAllDockerStats()
		if err != nil {
			// Docker может не быть установлен
			continue
		}
	}
}
