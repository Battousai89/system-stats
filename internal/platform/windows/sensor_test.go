package windows

import (
	"testing"

	"system-stats/internal/types"
)

func TestNewSensorTemperatures(t *testing.T) {
	sensors, err := NewSensorTemperatures()
	if err != nil {
		t.Fatalf("NewSensorTemperatures() returned error: %v", err)
	}

	if len(sensors) == 0 {
		t.Log("NewSensorTemperatures() returned empty slice (no thermal zones available)")
		return
	}

	for i, s := range sensors {
		if s.Name == "" {
			t.Errorf("Sensor[%d]: Name is empty", i)
		}

		// Температура должна быть в разумных пределах (-50 до +150)
		if s.Temperature < -50 || s.Temperature > 150 {
			t.Errorf("Sensor[%d]: Temperature %.1f°C is out of reasonable range", i, s.Temperature)
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

func TestGetSensorType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"CPU Thermal Zone", "CPU"},
		{"cpu_zone", "CPU"},
		{"GPU Temperature", "GPU"},
		{"graphics_thermal", "GPU"},
		{"Motherboard", "Motherboard"},
		{"MB Temp", "Motherboard"},
		{"System Zone", "Motherboard"},
		{"Ambient Temperature", "Ambient"},
		{"Battery Temp", "Battery"},
		{"Unknown Sensor", "Unknown"},
	}

	for _, test := range tests {
		result := getSensorType(test.name)
		if result != test.expected {
			t.Errorf("getSensorType(%q) = %s, expected %s", test.name, result, test.expected)
		}
	}
}

// Benchmark тесты
func BenchmarkNewSensorTemperatures(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewSensorTemperatures()
		if err != nil {
			b.Fatal(err)
		}
	}
}
