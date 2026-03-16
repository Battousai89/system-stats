package windows

import (
	"fmt"
	"runtime"
	"strings"

	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// win32OS структура для Win32_OperatingSystem
type win32OS struct {
	Caption         string `json:"Caption"`
	Version         string `json:"Version"`
	BuildNumber     string `json:"BuildNumber"`
	OSArchitecture  string `json:"OSArchitecture"`
	LastBootUpTime  string `json:"LastBootUpTime"`
}

// NewVirtualizationInfo получает информацию о виртуализации
func NewVirtualizationInfo() (*types.VirtualizationInfo, error) {
	info := &types.VirtualizationInfo{
		Platform:       "Windows",
		PlatformFamily: "Windows NT",
	}

	// Получаем информацию о системе
	script := `
		$cs = Get-CimInstance Win32_ComputerSystem | `+
			`Select-Object Manufacturer,Model,PCSystemType `+
		`$os = Get-CimInstance Win32_OperatingSystem | `+
			`Select-Object Caption,Version,BuildNumber,OSArchitecture,LastBootUpTime `+
		`@{cs=$cs;os=$os} | ConvertTo-Json -Depth 3`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return info, nil // Не считаем ошибкой
	}

	var result struct {
		CS *win32ComputerSystem `json:"cs"`
		OS *win32OS             `json:"os"`
	}
	if err := helpers.ParseJSON(string(output), &result); err != nil {
		return info, nil
	}

	// Заполняем информацию
	if result.OS != nil {
		info.PlatformVersion = result.OS.Caption
		info.Architecture = result.OS.OSArchitecture
		info.BootTime = parseWMIDateTimeToUnix(result.OS.LastBootUpTime)
	}

	if result.CS != nil {
		// Определяем виртуализацию по производителю
		manufacturer := result.CS.Manufacturer
		model := result.CS.Model

		info.Hypervisor, info.Virtualized = detectVirtualization(manufacturer, model)
		
		// Определяем тип гостевой ОС
		info.GuestType = detectGuestType(info.Hypervisor)
	}

	// Проверяем на контейнеры
	info.ContainerType = detectContainer()

	return info, nil
}

// detectVirtualization определяет тип виртуализации
func detectVirtualization(manufacturer, model string) (string, bool) {
	m := strings.ToLower(manufacturer)
	mod := strings.ToLower(model)

	// Hyper-V
	if m == "microsoft corporation" && strings.Contains(mod, "virtual") {
		return "Hyper-V", true
	}

	// VMware
	if strings.Contains(m, "vmware") {
		return "VMware", true
	}

	// VirtualBox
	if m == "innotek gmbh" || m == "oracle corporation" && strings.Contains(mod, "virtualbox") {
		return "VirtualBox", true
	}

	// QEMU/KVM
	if m == "qemu" || strings.Contains(mod, "kvm") {
		return "KVM", true
	}

	// Xen
	if strings.Contains(m, "xen") {
		return "Xen", true
	}

	// Amazon EC2
	if m == "amazon ec2" || strings.Contains(mod, "amazon") {
		return "Amazon EC2", true
	}

	// Google Cloud
	if m == "google compute engine" {
		return "Google Cloud", true
	}

	// DigitalOcean
	if m == "digitalocean" {
		return "DigitalOcean", true
	}

	return "", false
}

// detectGuestType определяет тип гостевой ОС
func detectGuestType(hypervisor string) string {
	switch hypervisor {
	case "Hyper-V":
		return "Windows VM"
	case "VMware":
		return "VMware Guest"
	case "VirtualBox":
		return "VirtualBox Guest"
	case "KVM":
		return "KVM Guest"
	case "Xen":
		return "Xen Guest"
	default:
		return ""
	}
}

// detectContainer определяет тип контейнера
func detectContainer() string {
	// Проверяем переменные окружения
	if isDockerContainer() {
		return "docker"
	}
	return ""
}

// isDockerContainer проверяет, запущены ли мы в Docker контейнере
func isDockerContainer() bool {
	// Простая проверка по наличию .dockerenv
	script := `Test-Path /.dockerenv`
	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "True"
}

// GetSystemInfo получает общую информацию о системе
func GetSystemInfo() (map[string]string, error) {
	result := make(map[string]string)
	result["GOOS"] = runtime.GOOS
	result["GOARCH"] = runtime.GOARCH
	result["NumCPU"] = fmt.Sprintf("%d", runtime.NumCPU())
	return result, nil
}
