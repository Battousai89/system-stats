//go:build linux
// +build linux

package linux

import (
	"testing"
)

func TestNewVirtualizationInfo(t *testing.T) {
	info, err := NewVirtualizationInfo()
	if err != nil {
		t.Fatalf("NewVirtualizationInfo() returned error: %v", err)
	}

	if info == nil {
		t.Fatal("NewVirtualizationInfo() returned nil")
	}

	if info.Platform != "Linux" {
		t.Errorf("NewVirtualizationInfo().Platform = %s, expected Linux", info.Platform)
	}

	if info.PlatformFamily != "Linux" {
		t.Errorf("NewVirtualizationInfo().PlatformFamily = %s, expected Linux", info.PlatformFamily)
	}

	t.Logf("Virtualization: Virtualized=%v, Hypervisor=%s, GuestType=%s, Container=%s",
		info.Virtualized, info.Hypervisor, info.GuestType, info.ContainerType)
	t.Logf("Architecture: %s, BootTime: %d", info.Architecture, info.BootTime)
}

func TestIsVirtualizedFromCPU(t *testing.T) {
	isVM := isVirtualizedFromCPU()
	t.Logf("Is virtualized (from CPU): %v", isVM)
}

func TestGetHypervisorFromDMI(t *testing.T) {
	hypervisor := getHypervisorFromDMI()
	if hypervisor != "" {
		t.Logf("Hypervisor from DMI: %s", hypervisor)
	} else {
		t.Log("No hypervisor detected from DMI (bare metal or undetectable)")
	}
}

func TestDetectContainer(t *testing.T) {
	container := detectContainer()
	if container != "" {
		t.Logf("Running in container: %s", container)
	} else {
		t.Log("Not running in a detected container")
	}
}

func TestDetectVirtualization(t *testing.T) {
	hypervisor, virtualized := detectVirtualization()
	t.Logf("Virtualization detected: %v, Hypervisor: %s", virtualized, hypervisor)
}

func TestDetectGuestType(t *testing.T) {
	tests := []struct {
		hypervisor string
		expected   string
	}{
		{"Hyper-V", "Windows VM"},
		{"VMware", "VMware Guest"},
		{"VirtualBox", "VirtualBox Guest"},
		{"KVM", "KVM Guest"},
		{"Xen", "Xen Guest"},
		{"QEMU", "QEMU Guest"},
		{"Unknown", "VM Guest"},
	}

	for _, test := range tests {
		result := detectGuestType(test.hypervisor)
		if result != test.expected {
			t.Errorf("detectGuestType(%q) = %s, expected %s", test.hypervisor, result, test.expected)
		}
	}
}

func TestGetPlatformVersion(t *testing.T) {
	version := getPlatformVersion()
	if version == "" {
		t.Log("Platform version is empty")
	} else {
		t.Logf("Platform version: %s", version)
	}
}

func TestGetSystemInfo(t *testing.T) {
	info, err := GetSystemInfo()
	if err != nil {
		t.Fatalf("GetSystemInfo() returned error: %v", err)
	}

	if info == nil {
		t.Fatal("GetSystemInfo() returned nil")
	}

	if info["GOOS"] != "linux" {
		t.Errorf("GetSystemInfo().GOOS = %s, expected linux", info["GOOS"])
	}

	if info["NumCPU"] == "" {
		t.Error("GetSystemInfo().NumCPU is empty")
	}

	t.Logf("System Info: GOOS=%s, GOARCH=%s, NumCPU=%s",
		info["GOOS"], info["GOARCH"], info["NumCPU"])
}

func TestVirtualizationInfoToPrint(t *testing.T) {
	info, err := NewVirtualizationInfo()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := info.ToPrint()
	if output == "" {
		t.Error("VirtualizationInfo.ToPrint() returned empty string")
	}

	t.Logf("VirtualizationInfo.ToPrint():\n%s", output)
}

func BenchmarkNewVirtualizationInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewVirtualizationInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDetectContainer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		detectContainer()
	}
}
