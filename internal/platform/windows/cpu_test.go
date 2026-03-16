package windows

import (
	"testing"
)

func TestNewCPUInfo(t *testing.T) {
	info, err := NewCPUInfo()
	if err != nil {
		t.Fatalf("NewCPUInfo() returned error: %v", err)
	}

	if len(info) == 0 {
		t.Fatal("NewCPUInfo() returned empty slice")
	}

	for i, cpu := range info {
		// Проверка обязательных полей
		if cpu.Name == "" {
			t.Errorf("CPU[%d]: Name is empty", i)
		}

		if cpu.Manufacturer == "" {
			t.Errorf("CPU[%d]: Manufacturer is empty", i)
		}

		// Проверка разумных значений
		if cpu.Cores == 0 {
			t.Errorf("CPU[%d]: Cores should be > 0, got %d", i, cpu.Cores)
		}

		if cpu.LogicalProcessors == 0 {
			t.Errorf("CPU[%d]: LogicalProcessors should be > 0, got %d", i, cpu.LogicalProcessors)
		}

		// Логических процессоров должно быть >= ядер
		if cpu.LogicalProcessors < cpu.Cores {
			t.Errorf("CPU[%d]: LogicalProcessors (%d) should be >= Cores (%d)", 
				i, cpu.LogicalProcessors, cpu.Cores)
		}

		if cpu.MaxClockSpeed == 0 {
			t.Errorf("CPU[%d]: MaxClockSpeed should be > 0", i)
		}

		// Текущая частота не должна превышать максимальную значительно
		if cpu.CurrentClockSpeed > cpu.MaxClockSpeed*2 {
			t.Errorf("CPU[%d]: CurrentClockSpeed (%d) is too high compared to MaxClockSpeed (%d)", 
				i, cpu.CurrentClockSpeed, cpu.MaxClockSpeed)
		}

		// LoadPercentage должен быть в диапазоне 0-100
		if cpu.LoadPercentage > 100 {
			t.Errorf("CPU[%d]: LoadPercentage should be <= 100, got %d", i, cpu.LoadPercentage)
		}

		// Температура (если есть) должна быть разумной
		if cpu.Temperature > 0 {
			if cpu.Temperature < 20 || cpu.Temperature > 150 {
				t.Errorf("CPU[%d]: Temperature (%d°C) is out of reasonable range", i, cpu.Temperature)
			}
		}

		// Кэш L3 должен быть >= L2 (обычно)
		if cpu.L2CacheSize > 0 && cpu.L3CacheSize > 0 {
			if cpu.L3CacheSize < cpu.L2CacheSize {
				t.Logf("CPU[%d]: Warning - L3CacheSize (%d) < L2CacheSize (%d)", 
					i, cpu.L3CacheSize, cpu.L2CacheSize)
			}
		}

		t.Logf("CPU[%d]: %s - %d cores, %d threads, %d MHz", 
			i, cpu.Name, cpu.Cores, cpu.LogicalProcessors, cpu.MaxClockSpeed)
	}
}

func TestNewCPUTimes(t *testing.T) {
	times, err := NewCPUTimes()
	if err != nil {
		t.Fatalf("NewCPUTimes() returned error: %v", err)
	}

	if len(times) == 0 {
		t.Fatal("NewCPUTimes() returned empty slice")
	}

	for i, tm := range times {
		// Проверка имени CPU
		if tm.CPU == "" {
			t.Errorf("CPUTimes[%d]: CPU name is empty", i)
		}

		// Все времена должны быть >= 0
		if tm.User < 0 {
			t.Errorf("CPUTimes[%d]: User time should be >= 0, got %f", i, tm.User)
		}
		if tm.System < 0 {
			t.Errorf("CPUTimes[%d]: System time should be >= 0, got %f", i, tm.System)
		}
		if tm.Idle < 0 {
			t.Errorf("CPUTimes[%d]: Idle time should be >= 0, got %f", i, tm.Idle)
		}
		if tm.Interrupt < 0 {
			t.Errorf("CPUTimes[%d]: Interrupt time should be >= 0, got %f", i, tm.Interrupt)
		}
		if tm.DPC < 0 {
			t.Errorf("CPUTimes[%d]: DPC time should be >= 0, got %f", i, tm.DPC)
		}

		// Usage должен быть в диапазоне 0-100
		if tm.Usage < 0 || tm.Usage > 100 {
			t.Errorf("CPUTimes[%d]: Usage should be 0-100, got %f", i, tm.Usage)
		}

		// Total должно быть > 0
		if tm.Total <= 0 {
			t.Errorf("CPUTimes[%d]: Total should be > 0", i)
		}

		t.Logf("CPUTimes[%d]: %s - User: %.4fs, System: %.4fs, Idle: %.4fs, Usage: %.2f%%",
			i, tm.CPU, tm.User, tm.System, tm.Idle, tm.Usage)
	}
}

func TestNewCPUPercent(t *testing.T) {
	percents, err := NewCPUPercent()
	if err != nil {
		t.Fatalf("NewCPUPercent() returned error: %v", err)
	}

	if len(percents) == 0 {
		t.Fatal("NewCPUPercent() returned empty slice")
	}

	for i, p := range percents {
		// Проверка имени CPU
		if p.CPU == "" {
			t.Errorf("CPUPercent[%d]: CPU name is empty", i)
		}

		// Все проценты должны быть в диапазоне 0-100
		if p.Percent < 0 || p.Percent > 100 {
			t.Errorf("CPUPercent[%d]: Percent should be 0-100, got %f", i, p.Percent)
		}
		if p.UserPercent < 0 || p.UserPercent > 100 {
			t.Errorf("CPUPercent[%d]: UserPercent should be 0-100, got %f", i, p.UserPercent)
		}
		if p.SystemPercent < 0 || p.SystemPercent > 100 {
			t.Errorf("CPUPercent[%d]: SystemPercent should be 0-100, got %f", i, p.SystemPercent)
		}
		if p.IdlePercent < 0 || p.IdlePercent > 100 {
			t.Errorf("CPUPercent[%d]: IdlePercent should be 0-100, got %f", i, p.IdlePercent)
		}

		// Сумма процентов должна быть примерно 100
		totalPercent := p.UserPercent + p.SystemPercent + p.IdlePercent
		if totalPercent < 95 || totalPercent > 105 {
			t.Logf("CPUPercent[%d]: Warning - total percent (%.2f) is not close to 100", i, totalPercent)
		}

		t.Logf("CPUPercent[%d]: %s - Total: %.2f%%, User: %.2f%%, System: %.2f%%, Idle: %.2f%%", 
			i, p.CPU, p.Percent, p.UserPercent, p.SystemPercent, p.IdlePercent)
	}
}

func TestGetCPUCoreCount(t *testing.T) {
	cores, err := GetCPUCoreCount()
	if err != nil {
		t.Fatalf("GetCPUCoreCount() returned error: %v", err)
	}

	if cores == 0 {
		t.Error("GetCPUCoreCount() returned 0 cores")
	}

	t.Logf("CPU Core Count: %d", cores)
}

func TestGetCPUThreadCount(t *testing.T) {
	threads, err := GetCPUThreadCount()
	if err != nil {
		t.Fatalf("GetCPUThreadCount() returned error: %v", err)
	}

	if threads == 0 {
		t.Error("GetCPUThreadCount() returned 0 threads")
	}

	t.Logf("CPU Thread Count: %d", threads)
}

func TestGetCPUModelName(t *testing.T) {
	name, err := GetCPUModelName()
	if err != nil {
		t.Fatalf("GetCPUModelName() returned error: %v", err)
	}

	if name == "" {
		t.Error("GetCPUModelName() returned empty name")
	}

	t.Logf("CPU Model Name: %s", name)
}

func TestGetAllCPUStats(t *testing.T) {
	stats, err := GetAllCPUStats()
	if err != nil {
		t.Fatalf("GetAllCPUStats() returned error: %v", err)
	}

	// Проверка Info
	if len(stats.Info) == 0 {
		t.Error("GetAllCPUStats().Info is empty")
	}

	// Проверка Times
	if len(stats.Times) == 0 {
		t.Error("GetAllCPUStats().Times is empty")
	}

	// Проверка Percent
	if len(stats.Percent) == 0 {
		t.Error("GetAllCPUStats().Percent is empty")
	}

	// Проверка CoreCount
	if stats.CoreCount == 0 {
		t.Error("GetAllCPUStats().CoreCount is 0")
	}

	// Проверка ThreadCount
	if stats.ThreadCount == 0 {
		t.Error("GetAllCPUStats().ThreadCount is 0")
	}

	// ThreadCount должен быть >= CoreCount
	if stats.ThreadCount < stats.CoreCount {
		t.Errorf("ThreadCount (%d) should be >= CoreCount (%d)", 
			stats.ThreadCount, stats.CoreCount)
	}

	t.Logf("AllCPUStats: %d CPUs, %d cores, %d threads", 
		len(stats.Info), stats.CoreCount, stats.ThreadCount)
}

func TestCPUInfoToPrint(t *testing.T) {
	info, err := NewCPUInfo()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := info[0].ToPrint()
	if output == "" {
		t.Error("CPUInfo.ToPrint() returned empty string")
	}

	t.Logf("CPUInfo.ToPrint():\n%s", output)
}

func TestCPUTimesToPrint(t *testing.T) {
	times, err := NewCPUTimes()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := times[0].ToPrint()
	if output == "" {
		t.Error("CPUTimes.ToPrint() returned empty string")
	}

	t.Logf("CPUTimes.ToPrint():\n%s", output)
}

func TestCPUPercentToPrint(t *testing.T) {
	percents, err := NewCPUPercent()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	output := percents[0].ToPrint()
	if output == "" {
		t.Error("CPUPercent.ToPrint() returned empty string")
	}

	t.Logf("CPUPercent.ToPrint():\n%s", output)
}

// Benchmark тесты
func BenchmarkNewCPUInfo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewCPUInfo()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewCPUTimes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewCPUTimes()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewCPUPercent(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewCPUPercent()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetAllCPUStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetAllCPUStats()
		if err != nil {
			b.Fatal(err)
		}
	}
}
