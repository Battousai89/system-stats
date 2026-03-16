package windows

import (
	"fmt"
	"strconv"
	"strings"

	"system-stats/internal/constants"
	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// win32VideoController структура для Win32_VideoController
type win32VideoController struct {
	Name            string `json:"Name"`
	Manufacturer    string `json:"Manufacturer"`
	DeviceID        string `json:"DeviceID"`
	PNPDeviceID     string `json:"PNPDeviceID"`
	VideoProcessor  string `json:"VideoProcessor"`
	DriverVersion   string `json:"DriverVersion"`
	AdapterRAM      uint64 `json:"AdapterRAM"`
	CurrentBitsPerPixel uint32 `json:"CurrentBitsPerPixel"`
	CurrentHorizontalResolution uint32 `json:"CurrentHorizontalResolution"`
	CurrentVerticalResolution   uint32 `json:"CurrentVerticalResolution"`
	CurrentRefreshRate uint32 `json:"CurrentRefreshRate"`
}

// win32DesktopMonitor структура для Win32_DesktopMonitor
type win32DesktopMonitor struct {
	MonitorManufacturer string `json:"MonitorManufacturer"`
	MonitorType         string `json:"MonitorType"`
}

// NewGPUInfo получает информацию о видеокартах
func NewGPUInfo() ([]types.GPUInfo, error) {
	// Получаем информацию о видеокартах
	script := `
		Get-CimInstance ` + constants.Win32VideoController + ` | `+
			`Select-Object Name,Manufacturer,DeviceID,PNPDeviceID,VideoProcessor,`+
			`DriverVersion,AdapterRAM,CurrentBitsPerPixel,`+
			`CurrentHorizontalResolution,CurrentVerticalResolution,`+
			`CurrentRefreshRate | `+
			`ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get GPU info: %w", err)
	}

	var videoControllers []win32VideoController
	if err := helpers.ParseJSON(string(output), &videoControllers); err != nil {
		var single win32VideoController
		if err2 := helpers.ParseJSON(string(output), &single); err2 == nil {
			videoControllers = []win32VideoController{single}
		} else {
			return nil, fmt.Errorf("failed to parse GPU info JSON: %w", err)
		}
	}

	result := make([]types.GPUInfo, 0, len(videoControllers))
	for _, vc := range videoControllers {
		// Пропускаем Microsoft Basic Render Driver
		if strings.Contains(vc.Name, "Microsoft Basic Render Driver") {
			continue
		}

		// Определяем производителя
		manufacturer := getGPUManufacturer(vc.Manufacturer, vc.PNPDeviceID)

		// Получаем название GPU по Device ID из PNPDeviceID
		deviceID := extractDeviceID(vc.PNPDeviceID)
		gpuName := types.GetGPUNameByDeviceID(deviceID)
		if gpuName == "" {
			gpuName = vc.Name
		}

		// Конвертируем AdapterRAM из байт в нормальное значение
		// Win32_VideoController возвращает байты, но может быть 0 для некоторых GPU
		memory := vc.AdapterRAM
		if memory == 0 {
			memory = extractMemoryFromName(vc.Name)
		}

		gpu := types.GPUInfo{
			Name:          gpuName,
			Manufacturer:  manufacturer,
			DeviceID:      deviceID,
			VendorID:      extractVendorID(vc.PNPDeviceID),
			DriverVersion: vc.DriverVersion,
			Memory:        memory,
			Resolution:    formatResolution(vc.CurrentHorizontalResolution, vc.CurrentVerticalResolution),
			RefreshRate:   uint8(vc.CurrentRefreshRate),
		}

		result = append(result, gpu)
	}

	return result, nil
}

// getGPUManufacturer определяет производителя GPU
func getGPUManufacturer(manufacturer string, deviceID string) string {
	if manufacturer != "" && manufacturer != "Microsoft Corporation" {
		return manufacturer
	}

	// Определяем по Vendor ID
	vendorID := extractVendorID(deviceID)
	switch vendorID {
	case "0x10DE":
		return "NVIDIA"
	case "0x1002", "0x1022":
		return "AMD"
	case "0x8086":
		return "Intel"
	}

	return "Unknown"
}

// extractDeviceID извлекает Device ID из строки
func extractDeviceID(deviceID string) string {
	// Формат: PCI\VEN_10DE&DEV_1B80&...
	parts := strings.Split(deviceID, "&")
	for _, part := range parts {
		if strings.HasPrefix(part, "DEV_") {
			return "0x" + part[4:]
		}
		// Также проверяем если VEN_ в середине части (PCI\VEN_...)
		if idx := strings.Index(part, "DEV_"); idx >= 0 {
			return "0x" + part[idx+4:idx+8]
		}
	}
	return deviceID
}

// extractVendorID извлекает Vendor ID из строки
func extractVendorID(deviceID string) string {
	// Формат: PCI\VEN_10DE&DEV_1B80&...
	parts := strings.Split(deviceID, "&")
	for _, part := range parts {
		if strings.HasPrefix(part, "VEN_") {
			return "0x" + part[4:8]
		}
		// Также проверяем если VEN_ в середине части (PCI\VEN_...)
		if idx := strings.Index(part, "VEN_"); idx >= 0 && idx+4 < len(part) {
			return "0x" + part[idx+4:idx+8]
		}
	}
	return ""
}

// extractMemoryFromName пытается извлечь объем памяти из названия
func extractMemoryFromName(name string) uint64 {
	nameLower := strings.ToLower(name)

	// Ищем паттерны типа "2GB", "4096MB", etc.
	if strings.Contains(nameLower, "gb") {
		for _, part := range strings.Fields(nameLower) {
			if strings.HasSuffix(part, "gb") {
				numStr := strings.TrimSuffix(part, "gb")
				if num, err := strconv.ParseFloat(numStr, 64); err == nil {
					return uint64(num * 1024 * 1024 * 1024)
				}
			}
		}
	}

	if strings.Contains(nameLower, "mb") {
		for _, part := range strings.Fields(nameLower) {
			if strings.HasSuffix(part, "mb") {
				numStr := strings.TrimSuffix(part, "mb")
				if num, err := strconv.ParseFloat(numStr, 64); err == nil {
					return uint64(num * 1024 * 1024)
				}
			}
		}
	}

	return 0
}

// formatResolution форматирует разрешение
func formatResolution(width, height uint32) string {
	if width == 0 || height == 0 {
		return ""
	}
	return fmt.Sprintf("%dx%d", width, height)
}
