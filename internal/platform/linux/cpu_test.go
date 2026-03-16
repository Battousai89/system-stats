//go:build linux
// +build linux

package linux

import (
	"testing"
)

func TestNewCPUInfo(t *testing.T) {
	info, err := NewCPUInfo()
	if err != nil {
		t.Fatalf("NewCPUInfo() returned error: %v", err)
	}

	if len(info) == 0 {
		t.Fatal("NewCPUInfo() returned empty slice")
	}

	for i, cpu := range info {
		if cpu.Name == "" {
			t.Errorf("CPU[%d]: Name is empty", i)
		}

		if cpu.Manufacturer == "" {
			t.Errorf("CPU[%d]: Manufacturer is empty", i)
		}

		if cpu.Cores == 0 {
			t.Errorf("CPU[%d]: Cores should be > 0, got %d", i, cpu.Cores)
		}

		if cpu.LogicalProcessors == 0 {
			t.Errorf("CPU[%d]: LogicalProcessors should be > 0, got %d", i, cpu.LogicalProcessors)
		}

		if cpu.LogicalProcessors < cpu.Cores {
			t.Errorf("CPU[%d]: LogicalProcessors (%d) should be >= Cores (%d)",
				i, cpu.LogicalProcessors, cpu.Cores)
		}

		if cpu.MaxClockSpeed == 0 {
			t.Errorf("CPU[%d]: MaxClockSpeed should be > 0", i)
		}

		if cpu.LoadPercentage > 100 {
			t.Errorf("CPU[%d]: LoadPercentage should be <= 100, got %d", i, cpu.LoadPercentage)
		}

		if cpu.Temperature > 0 {
			if cpu.Temperature < 20 || cpu.Temperature > 150 {
				t.Errorf("CPU[%d]: Temperature (%d°C) is out of reasonable range", i, cpu.Temperature)
			}
		}

		t.Logf("CPU[%d]: %s - %d cores, %d threads, %d MHz, %d°C",
			i, cpu.Name, cpu.Cores, cpu.LogicalProcessors, cpu.MaxClockSpeed, cpu.Temperature)
	}
}

func TestNewCPUTimes(t *testing.T) {
	times, err := NewCPUTimes()
	if err != nil {
		t.Fatalf("NewCPUTimes() returned error: %v", err)
	}

	if len(times) == 0 {
		t.Fatal("NewCPUTimes() returned empty slice")
	}

	for i, tm := range times {
		if tm.CPU == "" {
			t.Errorf("CPUTimes[%d]: CPU name is empty", i)
		}

		if tm.User < 0 {
			t.Errorf("CPUTimes[%d]: User time should be >= 0, got %f", i, tm.User)
		}
		if tm.System < 0 {
			t.Errorf("CPUTimes[%d]: System time should be >= 0, got %f", i, tm.System)
		}
		if tm.Idle < 0 {
			t.Errorf("CPUTimes[%d]: Idle time should be >= 0, got %f", i, tm.Idle)
		}
		if tm.Interrupt < 0 {
			t.Errorf("CPUTimes[%d]: Interrupt time should be >= 0, got %f", i, tm.Interrupt)
		}

		if tm.Usage < 0 || tm.Usage > 100 {
			t.Errorf("CPUTimes[%d]: Usage should be 0-100, got %f", i, tm.Usage)
		}

		if tm.Total <= 0 {
			t.Errorf("CPUTimes[%d]: Total should be > 0", i)
		}

		t.Logf("CPUTimes[%d]: %s - User: %.4fs, System: %.4fs, Idle: %.4fs, Usage: %.2f%%",
			i, tm.CPU, tm.User, tm.System, tm.Idle, tm.Usage)
	}
}

func TestNewCPUPercent(t *testing.T) {
	percents, err := NewCPUPercent()
	if err != nil {
		t.Fatalf("NewCPUPercent() returned error: %v", err)
	}

	if len(percents) == 0 {
		t.Fatal("NewCPUPercent() returned empty slice")
	}

	for i, p := range percents {
		if p.CPU == "" {
			t.Errorf("CPUPercent[%d]: CPU name is empty", i)
		}

		if p.Percent < 0 || p.Percent > 100 {
			t.Errorf("CPUPercent[%d]: Percent should be 0-100, got %f", i, p.Percent)
		}
		if p.UserPercent < 0 || p.UserPercent > 100 {
			t.Errorf("CPUPercent[%d]: UserPercent should be 0-100, got %f", i, p.UserPercent)
		}
		if p.SystemPercent < 0 || p.SystemPercent > 100 {
			t.Errorf("CPUPercent[%d]: SystemPercent should be 0-100, got %f", i, p.SystemPercent)
		}
		if p.IdlePercent < 0 || p.IdlePercent > 100 {
			t.Errorf("CPUPercent[%d]: IdlePercent should be 0-100, got %f", i, p.IdlePercent)
		}

		totalPercent := p.UserPercent + p.SystemPercent + p.IdlePercent
		if totalPercent < 90 || totalPercent > 110 {
			t.Logf("CPUPercent[%d]: Warning - total percent (%.2f) is not close to 100", i, totalPercent)
		}

		t.Logf("CPUPercent[%d]: %s - Total: %.2f%%, User: %.2f%%, System: %.2f%%, Idle: %.2f%%",
			i, p.CPU, p.Percent, p.UserPercent, p.SystemPercent, p.IdlePercent)
	}
}

func TestGetCPUCoreCount(t *testing.T) {
	cores, err := GetCPUCoreCount()
	if err != nil {
		t.Fatalf("GetCPUCoreCount() returned error: %v", err)
	}

	if cores == 0 {
		t.Error("GetCPUCoreCount() returned 0 cores")
	}

	t.Logf("CPU Core Count: %d", cores)
}

func TestGetCPUThreadCount(t *testing.T) {
	threads, err := GetCPUThreadCount()
	if err != nil {
		t.Fatalf("GetCPUThreadCount() returned error: %v", err)
	}

	if threads == 0 {
		t.Error("GetCPUThreadCount() returned 0 threads")
	}

	t.Logf("CPU Thread Count: %d", threads)
}

func TestGetCPUModelName(t *testing.T) {
	name, err := GetCPUModelName()
	if err != nil {
		t.Fatalf("GetCPUModelName() returned error: %v", err)
	}

	if name == "" {
		t.Error("GetCPUModelName() returned empty name")
	}

	t.Logf("CPU Model Name: %s", name)
}

func TestGetAllCPUStats(t *testing.T) {
	stats, err := GetAllCPUStats()
	if err != nil {
		t.Fatalf("GetAllCPUStats() returned error: %v", err)
	}

	if len(stats.Info) == 0 {
		t.Error("GetAllCPUStats().Info is empty")
	}

	if len(stats.Times) == 0 {
		t.Error("GetAllCPUStats().Times is empty")
	}

	if len(stats.Percent) == 0 {
		t.Error("GetAllCPUStats().Percent is empty")
	}

	if stats.CoreCount == 0 {
		t.Error("GetAllCPUStats().CoreCount is 0")
	}

	if stats.ThreadCount == 0 {
		t.Error("GetAllCPUStats().ThreadCount is 0")
	}

	if stats.ThreadCount < stats.CoreCount {
		t.Errorf("ThreadCount (%d) should be >= CoreCount (%d)",
			stats.ThreadCount, stats.CoreCount)
	}

	t.Logf("AllCPUStats: %d CPUs, %d cores, %d threads",
		len(stats.Info), stats.CoreCount, stats.ThreadCount)
}

func BenchmarkNewCPUInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewCPUInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewCPUTimes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewCPUTimes()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewCPUPercent(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewCPUPercent()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAllCPUStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAllCPUStats()
		if err != nil {
			b.Fatal(err)
		}
	}
}
