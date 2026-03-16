//go:build linux
// +build linux

package linux

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

	if info.Hostname == "" {
		t.Error("NewHostInfo().Hostname is empty")
	}

	if info.OS == "" {
		t.Error("NewHostInfo().OS is empty")
	}

	if info.Platform != "Linux" {
		t.Errorf("NewHostInfo().Platform = %s, expected Linux", info.Platform)
	}

	if info.KernelArch == "" {
		t.Error("NewHostInfo().KernelArch is empty")
	}

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
