//go:build linux
// +build linux

package linux

import (
	"testing"
)

func TestNewProcessInfo(t *testing.T) {
	processes, err := NewProcessInfo(10)
	if err != nil {
		t.Fatalf("NewProcessInfo() returned error: %v", err)
	}

	if processes == nil {
		t.Fatal("NewProcessInfo() returned nil")
	}

	if len(processes) == 0 {
		t.Log("No processes found")
		return
	}

	if len(processes) > 10 {
		t.Errorf("NewProcessInfo(10) returned %d processes, expected <= 10", len(processes))
	}

	for i, p := range processes {
		if p.Name == "" {
			t.Errorf("Process[%d]: Name is empty", i)
		}

		if p.PID == 0 {
			t.Errorf("Process[%d]: PID is 0", i)
		}

		if p.CPU < 0 {
			t.Errorf("Process[%d]: CPU should be >= 0, got %f", i, p.CPU)
		}

		t.Logf("Process[%d]: PID=%d, Name=%s, CPU=%.2f%%, Memory=%d MB",
			i, p.PID, p.Name, p.CPU, p.Memory/1024/1024)
	}
}

func TestGetProcessInfoByPID(t *testing.T) {
	// Get process 1 (init/systemd)
	info, err := GetProcessInfoByPID(1)
	if err != nil {
		t.Skipf("GetProcessInfoByPID(1) returned error: %v (may need root)", err)
	}

	if info == nil {
		t.Fatal("GetProcessInfoByPID(1) returned nil")
	}

	if info.PID != 1 {
		t.Errorf("GetProcessInfoByPID(1).PID = %d, expected 1", info.PID)
	}

	t.Logf("Process 1: Name=%s, CPU=%.2f%%, Memory=%d MB",
		info.Name, info.CPU, info.Memory/1024/1024)
}

func TestGetAllProcessStats(t *testing.T) {
	stats, err := GetAllProcessStats(5)
	if err != nil {
		t.Fatalf("GetAllProcessStats() returned error: %v", err)
	}

	if stats == nil {
		t.Fatal("GetAllProcessStats() returned nil")
	}

	if len(stats.Processes) == 0 {
		t.Log("No processes found")
		return
	}

	if len(stats.Processes) > 5 {
		t.Errorf("GetAllProcessStats(5) returned %d processes, expected <= 5", len(stats.Processes))
	}

	t.Logf("Process Stats: Total=%d, Showing top %d", stats.Total, len(stats.Processes))
}

func BenchmarkNewProcessInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewProcessInfo(10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAllProcessStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAllProcessStats(10)
		if err != nil {
			b.Fatal(err)
		}
	}
}
