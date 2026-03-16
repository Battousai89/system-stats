package windows

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

var (
	totalMemory      uint64
	totalMemoryOnce  sync.Once
	totalMemoryTime  time.Time
	memoryCacheTTL   = 60 * time.Second // Кэшируем на 1 минуту
)

// win32Process структура для Win32_Process
type win32Process struct {
	ProcessID      uint32 `json:"ProcessId"`
	Name           string `json:"Name"`
	ExecutablePath string `json:"ExecutablePath"`
	CommandLine    string `json:"CommandLine"`
	KernelModeTime uint64 `json:"KernelModeTime"`
	UserModeTime   uint64 `json:"UserModeTime"`
	WorkingSetSize uint64 `json:"WorkingSetSize"`
	PageFaults     uint32 `json:"PageFaults"`
	HandleCount    uint32 `json:"HandleCount"`
	ThreadCount    uint32 `json:"ThreadCount"`
	Status         string `json:"Status"`
	CreationDate   string `json:"CreationDate"`
}

// win32PerfProcess структура для счетчиков производительности
type win32PerfProcess struct {
	Name               string  `json:"Name"`
	PercentProcessorTime float64 `json:"PercentProcessorTime"`
	WorkingSet         uint64  `json:"WorkingSet"`
	IOReadBytesPerSec  float64 `json:"IOReadBytesPerSec"`
	IOWriteBytesPerSec float64 `json:"IOWriteBytesPerSec"`
}

// NewProcessInfo получает информацию о процессах
func NewProcessInfo(topN int) ([]types.ProcessInfo, error) {
	// Получаем список процессов
	procScript := `
		Get-CimInstance Win32_Process | `+
			`Select-Object ProcessId,Name,ExecutablePath,CommandLine,`+
			`WorkingSetSize,ThreadCount,CreationDate,Status | `+
			`ConvertTo-Json`

	procOutput, err := helpers.RunPowerShellCommand(procScript)
	if err != nil {
		return nil, fmt.Errorf("failed to get process list: %w", err)
	}

	var procs []win32Process
	if err := helpers.ParseJSON(string(procOutput), &procs); err != nil {
		var single win32Process
		if err2 := helpers.ParseJSON(string(procOutput), &single); err2 == nil {
			procs = []win32Process{single}
		} else {
			return nil, fmt.Errorf("failed to parse process JSON: %w", err)
		}
	}

	// Получаем CPU usage из perf counters
	perfScript := `
		Get-CimInstance Win32_PerfFormattedData_PerfProc_Process | `+
			`Where-Object { $_.Name -ne '_Total' -and -not $_.Name.StartsWith('#') } | `+
			`Select-Object Name,PercentProcessorTime,WorkingSet | `+
			`ConvertTo-Json`

	perfOutput, err := helpers.RunPowerShellCommand(perfScript)
	perfMap := make(map[string]float64)
	memMap := make(map[string]uint64)
	if err == nil {
		var perfs []win32PerfProcess
		if err2 := helpers.ParseJSON(string(perfOutput), &perfs); err2 == nil {
			for _, p := range perfs {
				name := normalizeProcessName(p.Name)
				perfMap[name] = p.PercentProcessorTime
				memMap[name] = p.WorkingSet
			}
		}
	}

	// Конвертируем процессы
	processes := make([]types.ProcessInfo, 0, len(procs))
	for _, wp := range procs {
		// Пропускаем системные процессы без имени
		if wp.Name == "" || wp.Name == "System Idle Process" {
			continue
		}

		// Получаем CPU из perf counters
		cpuPercent := perfMap[wp.Name]
		memory := wp.WorkingSetSize
		if memory == 0 {
			memory = memMap[wp.Name]
		}

		// Конвертируем время создания
		createTime := parseWMIDateTimeToUnix(wp.CreationDate)

		process := types.ProcessInfo{
			PID:         wp.ProcessID,
			Name:        wp.Name,
			CPU:         cpuPercent,
			Memory:      memory,
			MemoryPercent: 0,
			Status:      wp.Status,
			Cmdline:     wp.CommandLine,
			CreateTime:  createTime,
			NumThreads:  wp.ThreadCount,
		}

		processes = append(processes, process)
	}

	// Сортируем по CPU и берем топ N
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPU > processes[j].CPU
	})

	if topN > 0 && len(processes) > topN {
		processes = processes[:topN]
	}

	// Вычисляем процент памяти
	totalMemory := getTotalMemory()
	for i := range processes {
		if totalMemory > 0 {
			processes[i].MemoryPercent = float64(processes[i].Memory) / float64(totalMemory) * 100
		}
	}

	return processes, nil
}

// normalizeProcessName нормализует имя процесса для сопоставления
func normalizeProcessName(name string) string {
	// Убираем суффикс #PID который добавляет PerfProc_Process
	if idx := strings.Index(name, "#"); idx >= 0 {
		return name[:idx]
	}
	return name
}

// parseWMIDateTimeToUnix парсит WMI дату/время в unix timestamp
func parseWMIDateTimeToUnix(wmiTime string) uint64 {
	if wmiTime == "" {
		return 0
	}

	// Формат: 20240115123045.123456-480
	if len(wmiTime) < 14 {
		return 0
	}

	t, err := time.Parse("20060102150405", wmiTime[:14])
	if err != nil {
		return 0
	}

	return uint64(t.Unix())
}

// getTotalMemory получает общий объем памяти с кэшированием
func getTotalMemory() uint64 {
	// Проверяем кэш
	if totalMemory != 0 && time.Since(totalMemoryTime) < memoryCacheTTL {
		return totalMemory
	}

	// Получаем память через WMI (быстрый запрос без JSON)
	script := `(Get-CimInstance Win32_OperatingSystem).TotalVisibleMemorySize * 1024`
	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return 0
	}

	// Парсим число из вывода
	var result uint64
	fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &result)
	
	totalMemory = result
	totalMemoryTime = time.Now()
	return result
}

// GetProcessInfoByPID получает информацию о конкретном процессе
func GetProcessInfoByPID(pid uint32) (*types.ProcessInfo, error) {
	processes, err := NewProcessInfo(0)
	if err != nil {
		return nil, err
	}

	for _, p := range processes {
		if p.PID == pid {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("process %d not found", pid)
}
