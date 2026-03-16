package windows

import (
	"testing"

	"system-stats/internal/config"
	"system-stats/internal/types"
)

func TestNewProcessInfo(t *testing.T) {
	processes, err := NewProcessInfo(config.TopProcessesCount)
	if err != nil {
		t.Fatalf("NewProcessInfo() returned error: %v", err)
	}

	if len(processes) == 0 {
		t.Fatal("NewProcessInfo() returned empty slice")
	}

	for i, p := range processes {
		if p.Name == "" {
			t.Errorf("Process[%d]: Name is empty", i)
		}

		if p.PID == 0 {
			t.Errorf("Process[%d]: PID is 0", i)
		}

		if p.CPU < 0 {
			t.Errorf("Process[%d]: CPU %.1f < 0", i, p.CPU)
		}

		t.Logf("Process[%d]: PID=%d, Name=%s, CPU=%.1f%%, Memory=%d",
			i, p.PID, p.Name, p.CPU, p.Memory)
	}
}

func TestProcessInfosToPrint(t *testing.T) {
	processes, err := NewProcessInfo(10)
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := types.ProcessInfosToPrint(processes)
	if output == "" {
		t.Error("ProcessInfosToPrint() returned empty string")
	}

	t.Logf("ProcessInfosToPrint():\n%s", output)
}

func TestNormalizeProcessName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"chrome", "chrome"},
		{"chrome#1234", "chrome"},
		{"svchost#5678", "svchost"},
		{"#invalid", ""},
		{"", ""},
	}

	for _, test := range tests {
		result := normalizeProcessName(test.name)
		if result != test.expected {
			t.Errorf("normalizeProcessName(%q) = %q, expected %q", test.name, result, test.expected)
		}
	}
}

func TestParseWMIDateTimeToUnix(t *testing.T) {
	tests := []struct {
		wmiTime  string
		wantZero bool
	}{
		{"20240115123045.123456-480", false},
		{"20240115123045", false},
		{"", true},
		{"invalid", true},
		{"2024", true},
	}

	for _, test := range tests {
		result := parseWMIDateTimeToUnix(test.wmiTime)
		if test.wantZero && result != 0 {
			t.Errorf("parseWMIDateTimeToUnix(%q) = %d, expected 0", test.wmiTime, result)
		}
		if !test.wantZero && result == 0 {
			t.Errorf("parseWMIDateTimeToUnix(%q) = 0, expected non-zero", test.wmiTime)
		}
	}
}

func TestTruncateString(t *testing.T) {
	// truncateString не экспортируется, тестируем через ProcessInfosToPrint
	t.Skip("Internal function test skipped")
}

// Benchmark тесты
func BenchmarkNewProcessInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewProcessInfo(10)
		if err != nil {
			b.Fatal(err)
		}
	}
}
