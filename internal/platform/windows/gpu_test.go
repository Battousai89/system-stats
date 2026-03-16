package windows

import (
	"testing"

	"system-stats/internal/types"
)

func TestNewGPUInfo(t *testing.T) {
	gpus, err := NewGPUInfo()
	if err != nil {
		t.Fatalf("NewGPUInfo() returned error: %v", err)
	}

	if len(gpus) == 0 {
		t.Log("NewGPUInfo() returned empty slice (no dedicated GPU found)")
		return
	}

	for i, gpu := range gpus {
		if gpu.Name == "" {
			t.Errorf("GPU[%d]: Name is empty", i)
		}

		if gpu.Manufacturer == "" || gpu.Manufacturer == "Unknown" {
			t.Logf("GPU[%d]: Manufacturer is unknown", i)
		}

		t.Logf("GPU[%d]: %s - %s, Memory: %d bytes", i, gpu.Name, gpu.Manufacturer, gpu.Memory)
	}
}

func TestGetGPUManufacturer(t *testing.T) {
	tests := []struct {
		manufacturer string
		deviceID     string
		expected     string
	}{
		{"NVIDIA", "PCI\\VEN_10DE&DEV_1B80", "NVIDIA"},
		{"Advanced Micro Devices, Inc.", "PCI\\VEN_1002&DEV_67DF", "Advanced Micro Devices, Inc."},
		{"Intel Corporation", "PCI\\VEN_8086&DEV_5912", "Intel Corporation"},
		{"Microsoft Corporation", "PCI\\VEN_10DE&DEV_1B80", "NVIDIA"},
		{"Microsoft Corporation", "PCI\\VEN_1002&DEV_67DF", "AMD"},
		{"Microsoft Corporation", "PCI\\VEN_8086&DEV_5912", "Intel"},
		{"", "PCI\\VEN_10DE&DEV_1B80", "NVIDIA"},
		{"", "PCI\\VEN_1002&DEV_67DF", "AMD"},
		{"", "PCI\\VEN_8086&DEV_5912", "Intel"},
	}

	for _, test := range tests {
		result := getGPUManufacturer(test.manufacturer, test.deviceID)
		if result != test.expected {
			t.Errorf("getGPUManufacturer(%q, %q) = %s, expected %s",
				test.manufacturer, test.deviceID, result, test.expected)
		}
	}
}

func TestGPUInfosToPrint(t *testing.T) {
	gpus, err := NewGPUInfo()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	if len(gpus) == 0 {
		t.Skip("No GPUs to print")
	}

	output := types.GPUInfosToPrint(gpus)
	if output == "" {
		t.Error("GPUInfosToPrint() returned empty string")
	}

	t.Logf("GPUInfosToPrint():\n%s", output)
}

func TestExtractDeviceID(t *testing.T) {
	tests := []struct {
		deviceID string
		expected string
	}{
		{"PCI\\VEN_10DE&DEV_1B80&SUBSYS_1234", "0x1B80"},
		{"PCI\\VEN_1002&DEV_67DF&REV_00", "0x67DF"},
		{"PCI\\VEN_8086&DEV_5912", "0x5912"},
		{"Unknown", "Unknown"},
	}

	for _, test := range tests {
		result := extractDeviceID(test.deviceID)
		if result != test.expected {
			t.Errorf("extractDeviceID(%q) = %s, expected %s", test.deviceID, result, test.expected)
		}
	}
}

func TestExtractVendorID(t *testing.T) {
	tests := []struct {
		deviceID string
		expected string
	}{
		{"PCI\\VEN_10DE&DEV_1B80", "0x10DE"},
		{"PCI\\VEN_1002&DEV_67DF", "0x1002"},
		{"PCI\\VEN_8086&DEV_5912", "0x8086"},
		{"Unknown", ""},
	}

	for _, test := range tests {
		result := extractVendorID(test.deviceID)
		if result != test.expected {
			t.Errorf("extractVendorID(%q) = %s, expected %s", test.deviceID, result, test.expected)
		}
	}
}

func TestExtractMemoryFromName(t *testing.T) {
	tests := []struct {
		name     string
		expected uint64
	}{
		{"NVIDIA GeForce GTX 1060 6GB", 6 * 1024 * 1024 * 1024},
		{"AMD Radeon RX 580 8GB", 8 * 1024 * 1024 * 1024},
		{"Intel HD Graphics 4096MB", 4096 * 1024 * 1024},
		{"NVIDIA GeForce RTX 3080 10GB", 10 * 1024 * 1024 * 1024},
		{"Unknown GPU", 0},
	}

	for _, test := range tests {
		result := extractMemoryFromName(test.name)
		if result != test.expected {
			t.Errorf("extractMemoryFromName(%q) = %d, expected %d", test.name, result, test.expected)
		}
	}
}

func TestFormatResolution(t *testing.T) {
	tests := []struct {
		width    uint32
		height   uint32
		expected string
	}{
		{1920, 1080, "1920x1080"},
		{2560, 1440, "2560x1440"},
		{3840, 2160, "3840x2160"},
		{0, 0, ""},
		{1920, 0, ""},
	}

	for _, test := range tests {
		result := formatResolution(test.width, test.height)
		if result != test.expected {
			t.Errorf("formatResolution(%d, %d) = %s, expected %s",
				test.width, test.height, result, test.expected)
		}
	}
}

func TestGetGPUNameByDeviceID(t *testing.T) {
	tests := []struct {
		deviceID string
		expected string
	}{
		{"0x1B80", "NVIDIA GeForce GTX 1050 Ti"},
		{"0x67DF", "AMD Radeon RX 580/570"},
		{"0x5912", "Intel HD Graphics 630"},
		{"0x1234", ""},
	}

	for _, test := range tests {
		result := types.GetGPUNameByDeviceID(test.deviceID)
		if result != test.expected {
			t.Errorf("GetGPUNameByDeviceID(%q) = %s, expected %s", test.deviceID, result, test.expected)
		}
	}
}

// Benchmark тесты
func BenchmarkNewGPUInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewGPUInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}
