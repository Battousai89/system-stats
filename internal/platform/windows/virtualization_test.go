package windows

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

	if info.Platform == "" {
		t.Error("NewVirtualizationInfo().Platform is empty")
	}

	t.Logf("Virtualization: Virtualized=%v, Hypervisor=%s, Platform=%s",
		info.Virtualized, info.Hypervisor, info.Platform)
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

func TestDetectVirtualization(t *testing.T) {
	tests := []struct {
		manufacturer string
		model        string
		expectedHyp  string
		expectedVirt bool
	}{
		{"Microsoft Corporation", "Virtual Machine", "Hyper-V", true},
		{"VMware, Inc.", "VMware Virtual Platform", "VMware", true},
		{"innotek GmbH", "VirtualBox", "VirtualBox", true},
		{"QEMU", "Standard PC", "KVM", true},
		{"Dell Inc.", "PowerEdge R740", "", false},
		{"", "", "", false},
	}

	for _, test := range tests {
		hyp, virt := detectVirtualization(test.manufacturer, test.model)
		if hyp != test.expectedHyp || virt != test.expectedVirt {
			t.Errorf("detectVirtualization(%q, %q) = (%s, %v), expected (%s, %v)",
				test.manufacturer, test.model, hyp, virt, test.expectedHyp, test.expectedVirt)
		}
	}
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
		{"Unknown", ""},
		{"", ""},
	}

	for _, test := range tests {
		result := detectGuestType(test.hypervisor)
		if result != test.expected {
			t.Errorf("detectGuestType(%q) = %q, expected %q", test.hypervisor, result, test.expected)
		}
	}
}

func TestDetectContainer(t *testing.T) {
	// Эта функция зависит от окружения
	containerType := detectContainer()
	t.Logf("Container type: %q", containerType)
}

func TestIsDockerContainer(t *testing.T) {
	// Эта функция зависит от окружения
	isDocker := isDockerContainer()
	t.Logf("Is Docker container: %v", isDocker)
}

func TestGetSystemInfo(t *testing.T) {
	info, err := GetSystemInfo()
	if err != nil {
		t.Fatalf("GetSystemInfo() returned error: %v", err)
	}

	if info["GOOS"] == "" {
		t.Error("GetSystemInfo().GOOS is empty")
	}

	if info["GOARCH"] == "" {
		t.Error("GetSystemInfo().GOARCH is empty")
	}

	t.Logf("System Info: GOOS=%s, GOARCH=%s, NumCPU=%s",
		info["GOOS"], info["GOARCH"], info["NumCPU"])
}

// Benchmark тесты
func BenchmarkNewVirtualizationInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewVirtualizationInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}
