package windows

import (
	"strings"
	"testing"

	"system-stats/internal/types"
)

func TestNewDiskUsage(t *testing.T) {
	// Тестируем системный диск
	usage, err := NewDiskUsage("C:")
	if err != nil {
		t.Fatalf("NewDiskUsage(C:) returned error: %v", err)
	}

	if usage == nil {
		t.Fatal("NewDiskUsage(C:) returned nil")
	}

	// Проверка Total
	if usage.Total == 0 {
		t.Error("NewDiskUsage().Total is 0")
	}

	// Проверка Used
	if usage.Used == 0 {
		t.Error("NewDiskUsage().Used is 0")
	}

	if usage.Used > usage.Total {
		t.Errorf("Used (%d) should be <= Total (%d)", usage.Used, usage.Total)
	}

	// Проверка Free
	if usage.Free == 0 {
		t.Error("NewDiskUsage().Free is 0")
	}

	// Проверка Percent
	if usage.Percent < 0 || usage.Percent > 100 {
		t.Errorf("Percent should be 0-100, got %f", usage.Percent)
	}

	// Проверка согласованности
	expectedUsed := usage.Total - usage.Free
	if usage.Used != expectedUsed {
		t.Logf("Warning: Used (%d) != Total (%d) - Free (%d) = %d",
			usage.Used, usage.Total, usage.Free, expectedUsed)
	}

	t.Logf("Disk Usage C:: Total=%d, Used=%d, Free=%d, Percent=%.2f%%",
		usage.Total, usage.Used, usage.Free, usage.Percent)
}

func TestNewDiskIOCounters(t *testing.T) {
	counters, err := NewDiskIOCounters()
	if err != nil {
		t.Fatalf("NewDiskIOCounters() returned error: %v", err)
	}

	if len(counters) == 0 {
		t.Log("NewDiskIOCounters() returned empty slice (no disks found)")
		return
	}

	for i, c := range counters {
		// Проверка имени
		if c.Name == "" {
			t.Errorf("DiskIO[%d]: Name is empty", i)
		}

		t.Logf("DiskIO[%d]: %s - Read: %d B/s, Write: %d B/s",
			i, c.Name, c.ReadBytesPerSec, c.WriteBytesPerSec)
	}
}

func TestGetAllDiskDeviceInfo(t *testing.T) {
	devices, err := GetAllDiskDeviceInfo()
	if err != nil {
		t.Fatalf("GetAllDiskDeviceInfo() returned error: %v", err)
	}

	if len(devices) == 0 {
		t.Log("GetAllDiskDeviceInfo() returned empty slice (no physical disks found)")
		return
	}

	for i, d := range devices {
		// Проверка имени
		if d.Name == "" {
			t.Errorf("DiskDevice[%d]: Name is empty", i)
		}

		// Total может быть 0 для USB устройств
		if d.Total == 0 && !strings.Contains(d.Name, "USB") {
			t.Logf("DiskDevice[%d]: Total is 0 (may be normal for virtual/removable drives)", i)
		}

		// Проверка модели (может отсутствовать)
		if d.Model != "" {
			t.Logf("DiskDevice[%d]: %s - Model: %s, Size: %d, SSD: %v",
				i, d.Name, d.Model, d.Total, d.IsSSD)
		} else {
			t.Logf("DiskDevice[%d]: %s - Size: %d, SSD: %v",
				i, d.Name, d.Total, d.IsSSD)
		}
	}
}

func TestGetAllDiskStats(t *testing.T) {
	stats, err := GetAllDiskStats()
	if err != nil {
		t.Fatalf("GetAllDiskStats() returned error: %v", err)
	}

	if len(stats.Usage) == 0 {
		t.Error("GetAllDiskStats().Usage is empty")
	}

	t.Logf("Disk Stats: %d logical disks, %d IO counters, %d device info",
		len(stats.Usage), len(stats.IOCounters), len(stats.DeviceInfo))

	for i, u := range stats.Usage {
		t.Logf("Disk[%d]: %s - Total=%d, Used=%d, Free=%d, Percent=%.2f%%",
			i, u.Device, u.Total, u.Used, u.Free, u.Percent)
	}
}

func TestDiskUsageToPrint(t *testing.T) {
	usage, err := NewDiskUsage("C:")
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := usage.ToPrint()
	if output == "" {
		t.Error("DiskUsage.ToPrint() returned empty string")
	}

	t.Logf("DiskUsage.ToPrint():\n%s", output)
}

func TestDiskIOCountersToPrint(t *testing.T) {
	counters, err := NewDiskIOCounters()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	if len(counters) == 0 {
		t.Skip("No disk IO counters to print")
	}

	output := types.DiskIOCountersToPrint(counters)
	if output == "" {
		t.Error("DiskIOCountersToPrint() returned empty string")
	}

	t.Logf("DiskIOCountersToPrint():\n%s", output)
}

func TestDiskDeviceInfosToPrint(t *testing.T) {
	devices, err := GetAllDiskDeviceInfo()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	if len(devices) == 0 {
		t.Skip("No disk devices to print")
	}

	output := types.DiskDeviceInfosToPrint(devices)
	if output == "" {
		t.Error("DiskDeviceInfosToPrint() returned empty string")
	}

	t.Logf("DiskDeviceInfosToPrint():\n%s", output)
}

func TestIsSSDDrive(t *testing.T) {
	tests := []struct {
		model    string
		expected bool
	}{
		{"Samsung SSD 860 EVO", true},
		{"NVMe SSD", true},
		{"WD Blue SA510", true},
		{"Crucial MX500", true},
		{"ST1000DM003", false},
		{"Intel Optane", true},
		{"970 EVO Plus", true},
		{"ADATA SU635", true},
		{"Microsoft Storage Space Device", false},
		{"BR28 UDISK USB Device", false},
	}

	for _, test := range tests {
		result := isSSDDrive(test.model)
		if result != test.expected {
			t.Errorf("isSSDDrive(%q) = %v, expected %v", test.model, result, test.expected)
		}
	}
}

// Benchmark тесты
func BenchmarkNewDiskUsage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewDiskUsage("C:\\")
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

func BenchmarkGetAllDiskStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAllDiskStats()
		if err != nil {
			b.Fatal(err)
		}
	}
}
