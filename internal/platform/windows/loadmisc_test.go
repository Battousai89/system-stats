package windows

import (
	"testing"
)

func TestNewLoadMisc(t *testing.T) {
	info, err := NewLoadMisc()
	if err != nil {
		t.Fatalf("NewLoadMisc() returned error: %v", err)
	}

	if info == nil {
		t.Fatal("NewLoadMisc() returned nil")
	}

	t.Logf("LoadMisc: Procs=%d, Uptime=%ds, Load=%.2f/%.2f/%.2f",
		info.ProcsTotal, info.Uptime, info.Load1, info.Load5, info.Load15)
}

func TestLoadMiscToPrint(t *testing.T) {
	info, err := NewLoadMisc()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := info.ToPrint()
	if output == "" {
		t.Error("LoadMisc.ToPrint() returned empty string")
	}

	t.Logf("LoadMisc.ToPrint():\n%s", output)
}

func TestGetProcessCount(t *testing.T) {
	count, err := GetProcessCount()
	if err != nil {
		t.Fatalf("GetProcessCount() returned error: %v", err)
	}

	if count == 0 {
		t.Error("GetProcessCount() returned 0")
	}

	t.Logf("Process count: %d", count)
}

func TestGetThreadCount(t *testing.T) {
	count, err := GetThreadCount()
	if err != nil {
		t.Fatalf("GetThreadCount() returned error: %v", err)
	}

	if count == 0 {
		t.Error("GetThreadCount() returned 0")
	}

	t.Logf("Thread count: %d", count)
}

func TestGetUptime(t *testing.T) {
	uptime, err := GetUptime()
	if err != nil {
		t.Fatalf("GetUptime() returned error: %v", err)
	}

	if uptime == 0 {
		t.Error("GetUptime() returned 0")
	}

	t.Logf("Uptime: %ds (%.2f days)", uptime, float64(uptime)/86400.0)
}

func TestFormatUptimeFromHost(t *testing.T) {
	// formatUptimeFromHost находится в types package
	t.Skip("Internal function test skipped")
}

// Benchmark тесты
func BenchmarkNewLoadMisc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewLoadMisc()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetProcessCount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetProcessCount()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetUptime(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetUptime()
		if err != nil {
			b.Fatal(err)
		}
	}
}
