package windows

import (
	"testing"

	"system-stats/internal/types"
)

func TestGetVirtualMemory(t *testing.T) {
	mem, err := GetVirtualMemory()
	if err != nil {
		t.Fatalf("GetVirtualMemory() returned error: %v", err)
	}

	if mem == nil {
		t.Fatal("GetVirtualMemory() returned nil")
	}

	// Проверка Total
	if mem.Total == 0 {
		t.Error("GetVirtualMemory().Total is 0")
	}

	// Проверка Available
	if mem.Available > mem.Total {
		t.Errorf("Available (%d) should be <= Total (%d)", mem.Available, mem.Total)
	}

	// Проверка Used
	if mem.Used == 0 {
		t.Error("GetVirtualMemory().Used is 0")
	}

	if mem.Used > mem.Total {
		t.Errorf("Used (%d) should be <= Total (%d)", mem.Used, mem.Total)
	}

	// Проверка Free
	if mem.Free == 0 {
		t.Error("GetVirtualMemory().Free is 0")
	}

	// Проверка Percent
	if mem.Percent < 0 || mem.Percent > 100 {
		t.Errorf("Percent should be 0-100, got %f", mem.Percent)
	}

	// Проверка Committed
	if mem.Committed == 0 {
		t.Log("GetVirtualMemory().Committed is 0 (may be normal)")
	}

	// Проверка CommitLimit
	if mem.CommitLimit == 0 {
		t.Log("GetVirtualMemory().CommitLimit is 0 (may be normal)")
	}

	// Проверка согласованности
	if mem.Used+mem.Free != mem.Total {
		t.Logf("Warning: Used (%d) + Free (%d) != Total (%d)", mem.Used, mem.Free, mem.Total)
	}

	t.Logf("Memory: Total=%d, Available=%d, Used=%d, Free=%d, Percent=%.2f%%",
		mem.Total, mem.Available, mem.Used, mem.Free, mem.Percent)
}

func TestGetSwapDevices(t *testing.T) {
	swaps, err := GetSwapDevices()
	if err != nil {
		t.Fatalf("GetSwapDevices() returned error: %v", err)
	}

	// Swap может отсутствовать, это не ошибка
	if len(swaps) == 0 {
		t.Log("GetSwapDevices() returned empty slice (no swap configured)")
		return
	}

	for i, swap := range swaps {
		// Проверка имени
		if swap.Name == "" {
			t.Errorf("Swap[%d]: Name is empty", i)
		}

		// Проверка Total
		if swap.Total == 0 {
			t.Errorf("Swap[%d]: Total is 0", i)
		}

		// Проверка Used <= Total
		if swap.Used > swap.Total {
			t.Errorf("Swap[%d]: Used (%d) should be <= Total (%d)", i, swap.Used, swap.Total)
		}

		// Проверка Percent
		if swap.Percent < 0 || swap.Percent > 100 {
			t.Errorf("Swap[%d]: Percent should be 0-100, got %f", i, swap.Percent)
		}

		// Проверка Free
		expectedFree := swap.Total - swap.Used
		if swap.Free != expectedFree {
			t.Logf("Swap[%d]: Warning - Free (%d) != Total (%d) - Used (%d) = %d",
				i, swap.Free, swap.Total, swap.Used, expectedFree)
		}

		t.Logf("Swap[%d]: %s - Total=%d, Used=%d, Free=%d, Percent=%.2f%%",
			i, swap.Name, swap.Total, swap.Used, swap.Free, swap.Percent)
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
		t.Error("GetAllMemoryStats().Swap is nil (should be empty slice)")
	}

	if stats.Virtual != nil {
		t.Logf("Memory Stats: Total=%d, Available=%d, Used=%d, Percent=%.2f%%",
			stats.Virtual.Total, stats.Virtual.Available, stats.Virtual.Used, stats.Virtual.Percent)
	}

	if len(stats.Swap) > 0 {
		t.Logf("Swap devices: %d", len(stats.Swap))
	}
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

func TestSwapDevicesToPrint(t *testing.T) {
	swaps, err := GetSwapDevices()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	if len(swaps) == 0 {
		t.Skip("No swap devices to print")
	}

	output := types.SwapDevicesToPrint(swaps)
	if output == "" {
		t.Error("SwapDevicesToPrint() returned empty string")
	}

	t.Logf("SwapDevicesToPrint():\n%s", output)
}

// Benchmark тесты
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

func BenchmarkGetAllMemoryStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAllMemoryStats()
		if err != nil {
			b.Fatal(err)
		}
	}
}
