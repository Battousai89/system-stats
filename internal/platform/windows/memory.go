package windows

import (
	"fmt"

	"system-stats/internal/constants"
	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// win32OSMemory структура для парсинга вывода Win32_OperatingSystem (memory поля)
type win32OSMemory struct {
	TotalVisibleMemorySize     uint64 `json:"TotalVisibleMemorySize"`     // В KB
	FreePhysicalMemory         uint64 `json:"FreePhysicalMemory"`         // В KB
	TotalVirtualMemorySize     uint64 `json:"TotalVirtualMemorySize"`     // В KB
	FreeVirtualMemory          uint64 `json:"FreeVirtualMemory"`          // В KB
	CommittedPages             uint64 `json:"CommittedPages"`             // В KB
}

// win32PerfMemory структура для парсинга вывода Win32_PerfFormattedData_PerfOS_Memory
type win32PerfMemory struct {
	AvailableBytes             uint64 `json:"AvailableBytes"`
	AvailableKBytes            uint64 `json:"AvailableKBytes"`
	AvailableMBytes            uint64 `json:"AvailableMBytes"`
	CacheBytes                 uint64 `json:"CacheBytes"`
	CacheResidentBytes         uint64 `json:"CacheResidentBytes"`
	CommitLimit                uint64 `json:"CommitLimit"`
	CommittedBytes             uint64 `json:"CommittedBytes"`
	PoolNonpagedBytes          uint64 `json:"PoolNonpagedBytes"`
	PoolPagedBytes             uint64 `json:"PoolPagedBytes"`
	SystemCodeTotalBytes       uint64 `json:"SystemCodeTotalBytes"`
	SystemDriverTotalBytes     uint64 `json:"SystemDriverTotalBytes"`
	SystemCacheBytes           uint64 `json:"SystemCacheBytes"`
	TransitionFaultsPerSec     uint64 `json:"TransitionFaultsPerSec"`
	HardFaultsPerSec           uint64 `json:"HardFaultsPerSec"`
	PercentCommittedBytesInUse uint8  `json:"PercentCommittedBytesInUse"`
}

// win32PageFile структура для парсинга вывода Win32_PageFileUsage
type win32PageFile struct {
	Name            string `json:"Name"`
	AllocatedBaseSize uint64 `json:"AllocatedBaseSize"` // В MB
	CurrentUsage    uint64 `json:"CurrentUsage"`        // В MB
	PeakUsage       uint64 `json:"PeakUsage"`           // В MB
}

// GetVirtualMemory получает информацию об оперативной памяти
func GetVirtualMemory() (*types.VirtualMemory, error) {
	// Получаем данные из Win32_PerfFormattedData_PerfOS_Memory
	perfScript := `
		Get-CimInstance ` + constants.Win32PerfFormattedDataPerfOSMemory + ` | ` +
			`Select-Object AvailableBytes,CacheBytes,CommitLimit,CommittedBytes,` +
			`PoolNonpagedBytes,PoolPagedBytes,SystemCacheBytes,` +
			`PercentCommittedBytesInUse `+
			`| ConvertTo-Json`

	perfOutput, err := helpers.RunPowerShellCommand(perfScript)
	if err != nil {
		return nil, fmt.Errorf("failed to get perf memory info: %w", err)
	}

	var perfMem win32PerfMemory
	if err := helpers.ParseJSON(string(perfOutput), &perfMem); err != nil {
		return nil, fmt.Errorf("failed to parse perf memory JSON: %w", err)
	}

	// Получаем общую память из Win32_OperatingSystem
	osScript := `
		Get-CimInstance ` + constants.Win32OperatingSystem + ` | ` +
			`Select-Object TotalVisibleMemorySize,FreePhysicalMemory,` +
			`TotalVirtualMemorySize,FreeVirtualMemory `+
			`| ConvertTo-Json`

	osOutput, err := helpers.RunPowerShellCommand(osScript)
	if err != nil {
		return nil, fmt.Errorf("failed to get OS memory info: %w", err)
	}

	var osMem win32OSMemory
	if err := helpers.ParseJSON(string(osOutput), &osMem); err != nil {
		return nil, fmt.Errorf("failed to parse OS memory JSON: %w", err)
	}

	// Конвертируем KB в байты
	totalBytes := osMem.TotalVisibleMemorySize * 1024
	freeBytes := osMem.FreePhysicalMemory * 1024
	usedBytes := totalBytes - freeBytes

	// Вычисляем процент использования
	var percent float64
	if totalBytes > 0 {
		percent = float64(usedBytes) / float64(totalBytes) * 100.0
	}

	mem := &types.VirtualMemory{
		Total:       totalBytes,
		Available:   perfMem.AvailableBytes,
		Used:        usedBytes,
		Free:        freeBytes,
		Percent:     percent,
		Cached:      perfMem.CacheBytes,
		Committed:   perfMem.CommittedBytes,
		CommitLimit: perfMem.CommitLimit,
		Active:      perfMem.SystemCacheBytes,
		Buffers:     perfMem.PoolPagedBytes,
		Wired:       perfMem.PoolNonpagedBytes,
		PageFile:    osMem.TotalVirtualMemorySize * 1024,
	}

	return mem, nil
}

// NewVirtualMemory создает VirtualMemory из map (для совместимости)
func NewVirtualMemory(vm map[string]any) *types.VirtualMemory {
	return nil
}

// GetSwapDevices получает информацию о файлах подкачки
func GetSwapDevices() ([]types.SwapDevice, error) {
	script := `
		Get-CimInstance ` + constants.Win32PageFileUsage + ` | ` +
			`Select-Object Name,AllocatedBaseSize,CurrentUsage,PeakUsage ` +
			`| ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get swap info: %w", err)
	}

	var pageFiles []win32PageFile
	if err := helpers.ParseJSON(string(output), &pageFiles); err != nil {
		// Пробуем как одиночный объект
		var single win32PageFile
		if err2 := helpers.ParseJSON(string(output), &single); err2 == nil {
			pageFiles = []win32PageFile{single}
		} else {
			return nil, fmt.Errorf("failed to parse swap info JSON: %w", err)
		}
	}

	result := make([]types.SwapDevice, 0, len(pageFiles))
	for _, pf := range pageFiles {
		// Конвертируем MB в байты
		totalBytes := pf.AllocatedBaseSize * 1024 * 1024
		usedBytes := pf.CurrentUsage * 1024 * 1024
		freeBytes := totalBytes - usedBytes

		var percent float64
		if totalBytes > 0 {
			percent = float64(usedBytes) / float64(totalBytes) * 100.0
		}

		swap := types.SwapDevice{
			Name:        pf.Name,
			Total:       totalBytes,
			Used:        usedBytes,
			Free:        freeBytes,
			Percent:     percent,
			CurrentSize: usedBytes,
			PeakSize:    pf.PeakUsage * 1024 * 1024,
		}

		result = append(result, swap)
	}

	return result, nil
}

// NewSwapDevices создает список SwapDevice
func NewSwapDevices() ([]types.SwapDevice, error) {
	return GetSwapDevices()
}

// GetMemoryStats возвращает всю информацию о памяти в одной структуре
type MemoryStats struct {
	Virtual   *types.VirtualMemory  `json:"virtual"`
	Swap      []types.SwapDevice    `json:"swap"`
}

// GetAllMemoryStats собирает всю информацию о памяти
func GetAllMemoryStats() (*MemoryStats, error) {
	result := &MemoryStats{}

	vm, err := GetVirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory: %w", err)
	}
	result.Virtual = vm

	swap, err := GetSwapDevices()
	if err != nil {
		// Не считаем ошибку критичной для swap
		result.Swap = []types.SwapDevice{}
	} else {
		result.Swap = swap
	}

	return result, nil
}
