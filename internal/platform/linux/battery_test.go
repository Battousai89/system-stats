//go:build linux
// +build linux

package linux

import (
	"testing"

	"system-stats/internal/types"
)

func TestNewBatteryInfo(t *testing.T) {
	batteries, err := NewBatteryInfo()
	if err != nil {
		t.Fatalf("NewBatteryInfo() returned error: %v", err)
	}

	if batteries == nil {
		t.Fatal("NewBatteryInfo() returned nil")
	}

	if len(batteries) == 0 {
		t.Log("No batteries found (desktop system)")
		return
	}

	for i, b := range batteries {
		if b.Name == "" {
			t.Errorf("Battery[%d]: Name is empty", i)
		}

		if b.Percent > 100 {
			t.Errorf("Battery[%d]: Percent should be <= 100, got %d", i, b.Percent)
		}

		if b.DesignedCapacity == 0 {
			t.Errorf("Battery[%d]: DesignedCapacity is 0", i)
		}

		if b.FullChargeCap == 0 {
			t.Errorf("Battery[%d]: FullChargeCap is 0", i)
		}

		if b.CurrentCapacity > b.FullChargeCap {
			t.Errorf("Battery[%d]: CurrentCapacity (%d) > FullChargeCap (%d)",
				i, b.CurrentCapacity, b.FullChargeCap)
		}

		t.Logf("Battery[%d]: %s - %d%% (%d/%d mWh), Status=%s, Time=%ds",
			i, b.Name, b.Percent, b.CurrentCapacity, b.FullChargeCap, b.Status, b.EstTimeRemaining)
	}
}

func TestParseBattery(t *testing.T) {
	// Test parsing with mock data would require actual sysfs entries
	// This test just verifies the function doesn't crash
	batteries, err := NewBatteryInfo()
	if err != nil {
		t.Fatalf("parseBattery() returned error: %v", err)
	}

	t.Logf("Found %d batteries", len(batteries))
}

func TestGetAllBatteryStats(t *testing.T) {
	stats, err := GetAllBatteryStats()
	if err != nil {
		t.Fatalf("GetAllBatteryStats() returned error: %v", err)
	}

	if stats == nil {
		t.Fatal("GetAllBatteryStats() returned nil")
	}

	t.Logf("Battery Stats: Count=%d", stats.Count)

	if len(stats.Batteries) != stats.Count {
		t.Errorf("Battery count mismatch: len(Batteries)=%d, Count=%d",
			len(stats.Batteries), stats.Count)
	}
}

func TestBatteryInfosToPrint(t *testing.T) {
	batteries, err := NewBatteryInfo()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	if len(batteries) == 0 {
		t.Skip("No batteries to print")
	}

	output := types.BatteryInfosToPrint(batteries)
	if output == "" {
		t.Error("BatteryInfosToPrint() returned empty string")
	}

	t.Logf("BatteryInfosToPrint():\n%s", output)
}

func BenchmarkNewBatteryInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewBatteryInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAllBatteryStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAllBatteryStats()
		if err != nil {
			b.Fatal(err)
		}
	}
}
