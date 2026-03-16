package windows

import (
	"fmt"
	"time"

	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// win32PerfOS система для счетчиков ОС
type win32PerfOS struct {
	ContextSwitchesPerSec uint32 `json:"ContextSwitchesPerSec"`
	ProcessorQueueLength  uint32 `json:"ProcessorQueueLength"`
	SystemUpTime          uint32 `json:"SystemUpTime"`
	Processes             uint32 `json:"Processes"`
	Threads               uint32 `json:"Threads"`
	ExceptionDispatchesPerSec uint32 `json:"ExceptionDispatchesPerSec"`
}

// NewLoadMisc получает разную информацию о загрузке
func NewLoadMisc() (*types.LoadMisc, error) {
	info := &types.LoadMisc{}

	// Получаем счетчики производительности ОС
	script := `
		Get-CimInstance Win32_PerfFormattedData_PerfOS_System | `+
			`Select-Object ContextSwitchesPerSec,ProcessorQueueLength,SystemUpTime,`+
			`Processes,Threads,ExceptionDispatchesPerSec | `+
			`ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return info, nil // Не считаем ошибкой
	}

	var perfOS win32PerfOS
	if err := helpers.ParseJSON(string(output), &perfOS); err != nil {
		return info, nil
	}

	// Заполняем информацию
	info.ProcsTotal = uint64(perfOS.Processes)
	info.ProcsRunning = uint64(perfOS.ProcessorQueueLength)
	info.ContextSwitches = uint64(perfOS.ContextSwitchesPerSec)
	info.Interrupts = uint64(perfOS.ExceptionDispatchesPerSec)

	// SystemUpTime в секундах
	info.Uptime = uint64(perfOS.SystemUpTime)
	info.UptimeDays = float64(info.Uptime) / 86400.0

	// Вычисляем время загрузки
	if info.Uptime > 0 {
		info.BootTime = uint64(time.Now().Unix()) - info.Uptime
	}

	// Получаем load average (эмуляция через CPU usage)
	loadAvg, _ := NewLoadAvg()
	if loadAvg != nil {
		info.Load1 = loadAvg.Load1
		info.Load5 = loadAvg.Load5
		info.Load15 = loadAvg.Load15
	}

	return info, nil
}

// GetProcessCount получает количество процессов
func GetProcessCount() (uint32, error) {
	script := `
		Get-CimInstance Win32_Process | `+
			`Measure-Object | `+
			`Select-Object -ExpandProperty Count`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return 0, fmt.Errorf("failed to get process count: %w", err)
	}

	var count uint32
	fmt.Sscanf(string(output), "%d", &count)
	return count, nil
}

// GetThreadCount получает количество потоков
func GetThreadCount() (uint32, error) {
	script := `
		Get-CimInstance Win32_PerfFormattedData_PerfOS_System | `+
			`Select-Object Threads | `+
			`ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return 0, fmt.Errorf("failed to get thread count: %w", err)
	}

	var result struct {
		Threads uint32 `json:"Threads"`
	}
	if err := helpers.ParseJSON(string(output), &result); err != nil {
		return 0, fmt.Errorf("failed to parse thread count: %w", err)
	}

	return result.Threads, nil
}

// GetUptime получает время работы системы
func GetUptime() (uint64, error) {
	script := `
		Get-CimInstance Win32_PerfFormattedData_PerfOS_System | `+
			`Select-Object SystemUpTime | `+
			`ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return 0, fmt.Errorf("failed to get uptime: %w", err)
	}

	var result struct {
		SystemUpTime uint32 `json:"SystemUpTime"`
	}
	if err := helpers.ParseJSON(string(output), &result); err != nil {
		return 0, fmt.Errorf("failed to parse uptime: %w", err)
	}

	return uint64(result.SystemUpTime), nil
}
