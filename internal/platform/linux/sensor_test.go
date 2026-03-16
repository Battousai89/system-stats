//go:build linux
// +build linux

package linux

import (
	"testing"

	"system-stats/internal/types"
)

func TestNewSensorTemperatures(t *testing.T) {
	sensors, err := NewSensorTemperatures()
	if err != nil {
		t.Fatalf("NewSensorTemperatures() returned error: %v", err)
	}

	if sensors == nil {
		t.Fatal("NewSensorTemperatures() returned nil")
	}

	if len(sensors) == 0 {
		t.Log("No temperature sensors found (this is OK on some systems)")
		return
	}

	for i, s := range sensors {
		if s.Name == "" {
			t.Errorf("Sensor[%d]: Name is empty", i)
		}

		if s.Temperature < -50 || s.Temperature > 150 {
			t.Errorf("Sensor[%d]: Temperature (%.1f°C) is out of reasonable range", i, s.Temperature)
		}

		t.Logf("Sensor[%d]: %s (%s) - %.1f°C", i, s.Name, s.SensorType, s.Temperature)
	}
}

func TestSensorTemperaturesToPrint(t *testing.T) {
	sensors, err := NewSensorTemperatures()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	if len(sensors) == 0 {
		t.Skip("No sensors to print")
	}

	output := types.SensorTemperaturesToPrint(sensors)
	if output == "" {
		t.Error("SensorTemperaturesToPrint() returned empty string")
	}

	t.Logf("SensorTemperaturesToPrint():\n%s", output)
}

func BenchmarkNewSensorTemperatures(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewSensorTemperatures()
		if err != nil {
			b.Fatal(err)
		}
	}
}
