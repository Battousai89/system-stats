//go:build linux
// +build linux

package linux

import (
	"testing"

	"system-stats/internal/types"
)

func TestNewGPUInfo(t *testing.T) {
	gpus, err := NewGPUInfo()
	if err != nil {
		t.Fatalf("NewGPUInfo() returned error: %v", err)
	}

	if gpus == nil {
		t.Fatal("NewGPUInfo() returned nil")
	}

	if len(gpus) == 0 {
		t.Log("NewGPUInfo() returned empty slice (no dedicated GPU found)")
		return
	}

	for i, gpu := range gpus {
		// Allow empty name for integrated GPUs
		if gpu.Name == "" && gpu.Memory == 0 {
			t.Logf("GPU[%d]: Integrated GPU with no dedicated memory", i)
			continue
		}

		if gpu.Manufacturer == "" || gpu.Manufacturer == "Unknown" {
			t.Logf("GPU[%d]: Manufacturer is unknown", i)
		}

		t.Logf("GPU[%d]: %s - %s, Memory: %d MB", i, gpu.Name, gpu.Manufacturer, gpu.Memory/1024/1024)
	}
}

func TestGetGPUManufacturer(t *testing.T) {
	tests := []struct {
		vendorID string
		expected string
	}{
		{"0x10DE", "NVIDIA"},
		{"0x1002", "AMD"},
		{"0x1022", "AMD"},
		{"0x8086", "Intel"},
		{"0x13B5", "ARM"},
		{"0x5143", "Qualcomm"},
		{"0xFFFF", "Unknown"},
	}

	for _, test := range tests {
		result := getGPUManufacturer(test.vendorID)
		if result != test.expected {
			t.Errorf("getGPUManufacturer(%q) = %s, expected %s", test.vendorID, result, test.expected)
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

func TestExtractDeviceIDFromLspci(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"00:01.0 VGA compatible controller [0300]: Advanced Micro Devices, Inc. [AMD/ATI] [1002:98e4] (rev c6)", "0x98e4"},
		{"01:00.0 Display controller [0380]: Advanced Micro Devices, Inc. [AMD/ATI] [1002:6660] (rev 83)", "0x6660"},
		{"Invalid line without brackets", ""},
	}

	for _, test := range tests {
		result := extractDeviceIDFromLspci(test.line)
		if result != test.expected {
			t.Errorf("extractDeviceIDFromLspci(%q) = %s, expected %s", test.line, result, test.expected)
		}
	}
}

func TestExtractVendorIDFromLspci(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"00:01.0 VGA compatible controller [0300]: Advanced Micro Devices, Inc. [AMD/ATI] [1002:98e4] (rev c6)", "0x1002"},
		{"01:00.0 Display controller [0380]: Advanced Micro Devices, Inc. [AMD/ATI] [1002:6660] (rev 83)", "0x1002"},
		{"Invalid line without brackets", ""},
	}

	for _, test := range tests {
		result := extractVendorIDFromLspci(test.line)
		if result != test.expected {
			t.Errorf("extractVendorIDFromLspci(%q) = %s, expected %s", test.line, result, test.expected)
		}
	}
}

func TestNormalizeVendorID(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{"0x10de", "0x10de"},
		{"0x10DE", "0x10DE"},
		{"10de", "0x10de"},
		{"0x8086", "0x8086"},
	}

	for _, test := range tests {
		result := normalizeVendorID(test.id)
		if result != test.expected {
			t.Errorf("normalizeVendorID(%q) = %s, expected %s", test.id, result, test.expected)
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

func BenchmarkNewGPUInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewGPUInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}
