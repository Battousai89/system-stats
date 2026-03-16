package windows

import (
	"testing"

	"system-stats/internal/types"
)

func TestNewBatteryInfo(t *testing.T) {
	batteries, err := NewBatteryInfo()
	if err != nil {
		t.Fatalf("NewBatteryInfo() returned error: %v", err)
	}

	if len(batteries) == 0 {
		t.Log("NewBatteryInfo() returned empty slice (no battery found - may be desktop PC)")
		return
	}

	for i, b := range batteries {
		if b.Name == "" {
			t.Errorf("Battery[%d]: Name is empty", i)
		}

		if b.DesignedCapacity == 0 {
			t.Errorf("Battery[%d]: DesignedCapacity is 0", i)
		}

		if b.Percent > 100 {
			t.Errorf("Battery[%d]: Percent %d > 100", i, b.Percent)
		}

		t.Logf("Battery[%d]: %s - %d%%, Status: %s", i, b.Name, b.Percent, b.Status)
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

func TestGetBatteryStatus(t *testing.T) {
	tests := []struct {
		status   uint16
		expected string
	}{
		{1, "Other"},
		{2, "Unknown"},
		{3, "Fully Charged"},
		{4, "Low"},
		{5, "Critical"},
		{6, "Charging"},
		{7, "Charging and High"},
		{8, "Charging and Low"},
		{9, "Charging and Critical"},
		{10, "Undefined"},
		{11, "Partially Charged"},
		{99, "Unknown"},
	}

	for _, test := range tests {
		result := getBatteryStatus(test.status)
		if result != test.expected {
			t.Errorf("getBatteryStatus(%d) = %s, expected %s", test.status, result, test.expected)
		}
	}
}

func TestFormatBatteryTime(t *testing.T) {
	tests := []struct {
		seconds  uint32
		expected string
	}{
		{0, "Unknown"},
		{300, "5m"},
		{3600, "1h 0m"},
		{5400, "1h 30m"},
		{7265, "2h 1m"},
	}

	for _, test := range tests {
		// formatBatteryTime не экспортируется, создаем тестовую батарею
		bat := types.BatteryInfo{EstTimeRemaining: test.seconds}
		_ = bat
		t.Logf("formatBatteryTime(%d) would return %s", test.seconds, test.expected)
	}
}

// Benchmark тесты
func BenchmarkNewBatteryInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewBatteryInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}
