package windows

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"system-stats/internal/constants"
	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// win32ComputerSystem структура для Win32_ComputerSystem (используется в virtualization.go)
type win32ComputerSystem struct {
	Name                string `json:"Name"`
	Manufacturer        string `json:"Manufacturer"`
	Model               string `json:"Model"`
	TotalPhysicalMemory uint64 `json:"TotalPhysicalMemory"`
	PCSystemType        uint16 `json:"PCSystemType"`
}

// NewHostInfo получает информацию о хосте
func NewHostInfo() (*types.HostInfo, error) {
	// Объединенный запрос для OS и ComputerSystem
	script := `
		$os = Get-CimInstance ` + constants.Win32OperatingSystem + `
		$cs = Get-CimInstance ` + constants.Win32ComputerSystem + `
		[PSCustomObject]@{
			OSCaption = $os.Caption
			OSVersion = $os.Version
			OSBuildNumber = $os.BuildNumber
			OSLastBootUpTime = $os.LastBootUpTime
			OSArchitecture = $os.OSArchitecture
			OSProductType = $os.ProductType
			CSName = $cs.Name
			CSManufacturer = $cs.Manufacturer
			CSModel = $cs.Model
			CSTotalPhysicalMemory = $cs.TotalPhysicalMemory
		} | ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	var hostInfo struct {
		OSCaption           string `json:"OSCaption"`
		OSVersion           string `json:"OSVersion"`
		OSBuildNumber       string `json:"OSBuildNumber"`
		OSLastBootUpTime    string `json:"OSLastBootUpTime"`
		OSArchitecture      string `json:"OSArchitecture"`
		OSProductType       uint32 `json:"OSProductType"`
		CSName              string `json:"CSName"`
		CSManufacturer      string `json:"CSManufacturer"`
		CSModel             string `json:"CSModel"`
		CSTotalPhysicalMemory uint64 `json:"CSTotalPhysicalMemory"`
	}

	if err := helpers.ParseJSON(string(output), &hostInfo); err != nil {
		return nil, fmt.Errorf("failed to parse host info JSON: %w", err)
	}

	// Получаем uptime
	var uptime uint64
	bootTime, err := parseWMIDateTime(hostInfo.OSLastBootUpTime)
	if err == nil {
		uptime = uint64(time.Since(bootTime).Seconds())
	}

	// Определяем роль
	role := "Unknown"
	switch hostInfo.OSProductType {
	case 1:
		role = "Workstation"
	case 2:
		role = "Domain Controller"
	case 3:
		role = "Server"
	}

	// Определяем виртуализацию по производителю
	virtualization := ""
	if hostInfo.CSManufacturer != "" {
		manufacturer := hostInfo.CSManufacturer
		if isVirtualMachine(manufacturer) {
			virtualization = manufacturer
		}
	}

	info := &types.HostInfo{
		Hostname:        getHostname(),
		Uptime:          uptime,
		OS:              hostInfo.OSCaption,
		Platform:        "Windows",
		PlatformFamily:  "Windows NT",
		PlatformVersion: hostInfo.OSVersion,
		KernelVersion:   hostInfo.OSBuildNumber,
		KernelArch:      hostInfo.OSArchitecture,
		Virtualization:  virtualization,
		Role:            role,
	}

	return info, nil
}

// parseWMIDateTime парсит дату/время в формате WMI
func parseWMIDateTime(wmiTime string) (time.Time, error) {
	if wmiTime == "" {
		return time.Time{}, fmt.Errorf("empty WMI time")
	}

	// WMI формат: 20240115123045.123456-480
	// Обрезаем до 14 символов: 20240115123045
	if len(wmiTime) < 14 {
		return time.Time{}, fmt.Errorf("invalid WMI time format: %s", wmiTime)
	}

	timeStr := wmiTime[:14]
	return time.Parse("20060102150405", timeStr)
}

// isVirtualMachine определяет, является ли машина виртуальной
func isVirtualMachine(manufacturer string) bool {
	virtualManufacturers := []string{
		"Microsoft Corporation", // Hyper-V
		"VMware, Inc.",
		"VMware Virtual Platform",
		"Xen",
		"QEMU",
		"VirtualBox",
		"innotek GmbH",
		"Amazon EC2",
		"Google Compute Engine",
		"DigitalOcean",
	}

	for _, vm := range virtualManufacturers {
		if manufacturer == vm {
			return true
		}
	}

	return false
}

// getHostname получает имя хоста
func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// NewLoadAvg получает среднюю загрузку (для Windows эмулируется через CPU usage)
// Windows не имеет понятия load average как в Unix
func NewLoadAvg() (*types.LoadAvg, error) {
	// Для Windows используем 1-минутный средний CPU usage как approximation
	script := `
		Get-CimInstance ` + constants.Win32PerfFormattedDataPerfOSProcessor + ` | `+
			`Where-Object { $_.Name -eq '_Total' } | `+
			`Select-Object PercentProcessorTime | `+
			`ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		// Возвращаем нулевые значения если не удалось получить
		return &types.LoadAvg{
			Load1:  0,
			Load5:  0,
			Load15: 0,
		}, nil
	}

	var result struct {
		PercentProcessorTime float64 `json:"PercentProcessorTime"`
	}
	if err := helpers.ParseJSON(string(output), &result); err != nil {
		return &types.LoadAvg{
			Load1:  0,
			Load5:  0,
			Load15: 0,
		}, nil
	}

	// Конвертируем процент CPU в load average (очень грубая аппроксимация)
	// Load average = CPU usage / 100 * количество ядер
	cpuCount := runtime.NumCPU()
	load := result.PercentProcessorTime / 100.0 * float64(cpuCount)

	return &types.LoadAvg{
		Load1:  load,
		Load5:  load, // Для Windows все значения одинаковые
		Load15: load,
	}, nil
}
