//go:build linux
// +build linux

package linux

import (
	"testing"
)

func TestNewDiskUsage(t *testing.T) {
	usage, err := NewDiskUsage("/")
	if err != nil {
		t.Fatalf("NewDiskUsage('/') returned error: %v", err)
	}

	if usage == nil {
		t.Fatal("NewDiskUsage() returned nil")
	}

	if usage.Total == 0 {
		t.Error("NewDiskUsage().Total is 0")
	}

	if usage.Used > usage.Total {
		t.Errorf("Used (%d) should be <= Total (%d)", usage.Used, usage.Total)
	}

	if usage.Percent < 0 || usage.Percent > 100 {
		t.Errorf("NewDiskUsage().Percent should be 0-100, got %f", usage.Percent)
	}

	t.Logf("Disk Usage: Total=%d GB, Used=%d GB, Free=%d GB (%.2f%%)",
		usage.Total/1024/1024/1024, usage.Used/1024/1024/1024, usage.Free/1024/1024/1024, usage.Percent)
}

func TestNewDiskIOCounters(t *testing.T) {
	counters, err := NewDiskIOCounters()
	if err != nil {
		t.Fatalf("NewDiskIOCounters() returned error: %v", err)
	}

	if counters == nil {
		t.Fatal("NewDiskIOCounters() returned nil")
	}

	if len(counters) == 0 {
		t.Log("No disk I/O counters found")
		return
	}

	for i, c := range counters {
		if c.Name == "" {
			t.Errorf("DiskIO[%d]: Name is empty", i)
		}

		t.Logf("DiskIO[%d]: %s - Read=%d MB, Write=%d MB",
			i, c.Name, c.ReadBytes/1024/1024, c.WriteBytes/1024/1024)
	}
}

func TestGetAllDiskDeviceInfo(t *testing.T) {
	devices, err := GetAllDiskDeviceInfo()
	if err != nil {
		t.Fatalf("GetAllDiskDeviceInfo() returned error: %v", err)
	}

	if devices == nil {
		t.Fatal("GetAllDiskDeviceInfo() returned nil")
	}

	if len(devices) == 0 {
		t.Log("No disk devices found")
		return
	}

	for i, dev := range devices {
		if dev.Name == "" {
			t.Errorf("DiskDevice[%d]: Name is empty", i)
		}

		if dev.Total == 0 {
			t.Errorf("DiskDevice[%d]: Total is 0", i)
		}

		t.Logf("DiskDevice[%d]: %s - Model=%s, Total=%d GB, SSD=%v",
			i, dev.Name, dev.Model, dev.Total/1024/1024/1024, dev.IsSSD)
	}
}

func TestDiskUsageToPrint(t *testing.T) {
	usage, err := NewDiskUsage("/")
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := usage.ToPrint()
	if output == "" {
		t.Error("DiskUsage.ToPrint() returned empty string")
	}

	t.Logf("DiskUsage.ToPrint():\n%s", output)
}

func BenchmarkNewDiskUsage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewDiskUsage("/")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewDiskIOCounters(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewDiskIOCounters()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAllDiskDeviceInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAllDiskDeviceInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}
