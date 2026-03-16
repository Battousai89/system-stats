package windows

import (
	"testing"
)

func TestNewHostInfo(t *testing.T) {
	info, err := NewHostInfo()
	if err != nil {
		t.Fatalf("NewHostInfo() returned error: %v", err)
	}

	if info == nil {
		t.Fatal("NewHostInfo() returned nil")
	}

	// Проверка hostname
	if info.Hostname == "" {
		t.Error("NewHostInfo().Hostname is empty")
	}

	// Проверка OS
	if info.OS == "" {
		t.Error("NewHostInfo().OS is empty")
	}

	// Проверка Platform
	if info.Platform != "Windows" {
		t.Errorf("NewHostInfo().Platform = %s, expected Windows", info.Platform)
	}

	// Проверка KernelArch
	if info.KernelArch == "" {
		t.Error("NewHostInfo().KernelArch is empty")
	}

	// Проверка uptime
	if info.Uptime == 0 {
		t.Log("NewHostInfo().Uptime is 0 (system just booted)")
	}

	t.Logf("HostInfo: Hostname=%s, OS=%s, Platform=%s, Arch=%s, Uptime=%ds",
		info.Hostname, info.OS, info.Platform, info.KernelArch, info.Uptime)
}

func TestNewLoadAvg(t *testing.T) {
	load, err := NewLoadAvg()
	if err != nil {
		t.Fatalf("NewLoadAvg() returned error: %v", err)
	}

	if load == nil {
		t.Fatal("NewLoadAvg() returned nil")
	}

	// Load average может быть 0 если CPU idle
	t.Logf("LoadAvg: 1min=%.2f, 5min=%.2f, 15min=%.2f",
		load.Load1, load.Load5, load.Load15)
}

func TestHostInfoToPrint(t *testing.T) {
	info, err := NewHostInfo()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := info.ToPrint()
	if output == "" {
		t.Error("HostInfo.ToPrint() returned empty string")
	}

	t.Logf("HostInfo.ToPrint():\n%s", output)
}

func TestLoadAvgToPrint(t *testing.T) {
	load, err := NewLoadAvg()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := load.ToPrint()
	if output == "" {
		t.Error("LoadAvg.ToPrint() returned empty string")
	}

	t.Logf("LoadAvg.ToPrint():\n%s", output)
}

func TestIsVirtualMachine(t *testing.T) {
	tests := []struct {
		manufacturer string
		expected     bool
	}{
		{"Microsoft Corporation", true},
		{"VMware, Inc.", true},
		{"Xen", true},
		{"QEMU", true},
		{"VirtualBox", true},
		{"Amazon EC2", true},
		{"Google Compute Engine", true},
		{"Dell Inc.", false},
		{"HP", false},
		{"Lenovo", false},
		{"ASUS", false},
	}

	for _, test := range tests {
		result := isVirtualMachine(test.manufacturer)
		if result != test.expected {
			t.Errorf("isVirtualMachine(%q) = %v, expected %v", test.manufacturer, result, test.expected)
		}
	}
}

func TestParseWMIDateTime(t *testing.T) {
	tests := []struct {
		wmiTime   string
		wantError bool
	}{
		{"20240115123045.123456-480", false},
		{"20240115123045", false},
		{"", true},
		{"invalid", true},
		{"2024", true},
	}

	for _, test := range tests {
		_, err := parseWMIDateTime(test.wmiTime)
		if (err != nil) != test.wantError {
			t.Errorf("parseWMIDateTime(%q) error = %v, wantError %v", test.wmiTime, err, test.wantError)
		}
	}
}

// Benchmark тесты
func BenchmarkNewHostInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewHostInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewLoadAvg(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewLoadAvg()
		if err != nil {
			b.Fatal(err)
		}
	}
}
