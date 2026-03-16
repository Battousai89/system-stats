//go:build linux
// +build linux

package linux

import (
	"testing"
)

func TestGetVirtualMemory(t *testing.T) {
	mem, err := GetVirtualMemory()
	if err != nil {
		t.Fatalf("GetVirtualMemory() returned error: %v", err)
	}

	if mem == nil {
		t.Fatal("GetVirtualMemory() returned nil")
	}

	if mem.Total == 0 {
		t.Error("GetVirtualMemory().Total is 0")
	}

	if mem.Available == 0 {
		t.Error("GetVirtualMemory().Available is 0")
	}

	if mem.Used == 0 {
		t.Error("GetVirtualMemory().Used is 0")
	}

	if mem.Percent < 0 || mem.Percent > 100 {
		t.Errorf("GetVirtualMemory().Percent should be 0-100, got %f", mem.Percent)
	}

	if mem.Available > mem.Total {
		t.Errorf("Available (%d) should be <= Total (%d)", mem.Available, mem.Total)
	}

	if mem.Used > mem.Total {
		t.Errorf("Used (%d) should be <= Total (%d)", mem.Used, mem.Total)
	}

	t.Logf("Memory: Total=%d MB, Available=%d MB, Used=%d MB (%.2f%%)",
		mem.Total/1024/1024, mem.Available/1024/1024, mem.Used/1024/1024, mem.Percent)
}

func TestGetSwapDevices(t *testing.T) {
	swaps, err := GetSwapDevices()
	if err != nil {
		t.Fatalf("GetSwapDevices() returned error: %v", err)
	}

	if swaps == nil {
		t.Fatal("GetSwapDevices() returned nil")
	}

	if len(swaps) > 0 {
		for i, swap := range swaps {
			if swap.Name == "" {
				t.Errorf("Swap[%d]: Name is empty", i)
			}

			if swap.Total == 0 {
				t.Errorf("Swap[%d]: Total is 0", i)
			}

			if swap.Percent < 0 || swap.Percent > 100 {
				t.Errorf("Swap[%d]: Percent should be 0-100, got %f", i, swap.Percent)
			}

			t.Logf("Swap[%d]: %s - Total=%d MB, Used=%d MB (%.2f%%)",
				i, swap.Name, swap.Total/1024/1024, swap.Used/1024/1024, swap.Percent)
		}
	} else {
		t.Log("No swap devices found (this is OK)")
	}
}

func TestGetAllMemoryStats(t *testing.T) {
	stats, err := GetAllMemoryStats()
	if err != nil {
		t.Fatalf("GetAllMemoryStats() returned error: %v", err)
	}

	if stats.Virtual == nil {
		t.Error("GetAllMemoryStats().Virtual is nil")
	}

	if stats.Swap == nil {
		t.Error("GetAllMemoryStats().Swap is nil")
	}

	t.Logf("Memory Stats: Total=%d MB, Swap devices=%d",
		stats.Virtual.Total/1024/1024, len(stats.Swap))
}

func TestVirtualMemoryToPrint(t *testing.T) {
	mem, err := GetVirtualMemory()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := mem.ToPrint()
	if output == "" {
		t.Error("VirtualMemory.ToPrint() returned empty string")
	}

	t.Logf("VirtualMemory.ToPrint():\n%s", output)
}

func BenchmarkGetVirtualMemory(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetVirtualMemory()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetSwapDevices(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetSwapDevices()
		if err != nil {
			b.Fatal(err)
		}
	}
}
