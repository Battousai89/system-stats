//go:build linux
// +build linux

package linux

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

	if info.ProcsTotal == 0 {
		t.Error("NewLoadMisc().ProcsTotal is 0")
	}

	if info.Uptime == 0 {
		t.Error("NewLoadMisc().Uptime is 0")
	}

	if info.UptimeDays < 0 {
		t.Errorf("NewLoadMisc().UptimeDays should be >= 0, got %f", info.UptimeDays)
	}

	if info.BootTime == 0 {
		t.Error("NewLoadMisc().BootTime is 0")
	}

	t.Logf("LoadMisc: Procs=%d, Running=%d, Uptime=%ds (%.2f days), BootTime=%d",
		info.ProcsTotal, info.ProcsRunning, info.Uptime, info.UptimeDays, info.BootTime)
	t.Logf("Load: %.2f, %.2f, %.2f | ContextSwitches=%d, Interrupts=%d",
		info.Load1, info.Load5, info.Load15, info.ContextSwitches, info.Interrupts)
}

func TestGetUptime(t *testing.T) {
	uptime, err := getUptime()
	if err != nil {
		t.Fatalf("getUptime() returned error: %v", err)
	}

	if uptime == 0 {
		t.Error("getUptime() returned 0")
	}

	t.Logf("Uptime: %d seconds (%.2f days)", uptime, float64(uptime)/86400.0)
}

func TestGetBootTime(t *testing.T) {
	bootTime := getBootTime()
	if bootTime == 0 {
		t.Error("getBootTime() returned 0")
	}

	t.Logf("BootTime: %d", bootTime)
}

func TestGetLoadAverage(t *testing.T) {
	load, err := getLoadAverage()
	if err != nil {
		t.Fatalf("getLoadAverage() returned error: %v", err)
	}

	if load == nil {
		t.Fatal("getLoadAverage() returned nil")
	}

	t.Logf("Load Average: 1min=%.2f, 5min=%.2f, 15min=%.2f",
		load.Load1, load.Load5, load.Load15)
}

func TestGetProcessCount(t *testing.T) {
	count, err := GetProcessCount()
	if err != nil {
		t.Fatalf("GetProcessCount() returned error: %v", err)
	}

	if count == 0 {
		t.Error("GetProcessCount() returned 0")
	}

	t.Logf("Process Count: %d", count)
}

func TestGetThreadCount(t *testing.T) {
	count, err := GetThreadCount()
	if err != nil {
		t.Fatalf("GetThreadCount() returned error: %v", err)
	}

	if count == 0 {
		t.Error("GetThreadCount() returned 0")
	}

	t.Logf("Thread Count: %d", count)
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

func BenchmarkGetThreadCount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetThreadCount()
		if err != nil {
			b.Fatal(err)
		}
	}
}
