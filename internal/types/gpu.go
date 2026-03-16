package types

import (
	"fmt"
	"strings"

	"system-stats/internal/constants"
	"system-stats/internal/helpers"
)

// GPUInfo информация о видеокарте
type GPUInfo struct {
	Name          string  `json:"name"`           // Название GPU
	Manufacturer  string  `json:"manufacturer"`   // Производитель (NVIDIA, AMD, Intel)
	DeviceID      string  `json:"device_id"`      // Device ID
	VendorID      string  `json:"vendor_id"`      // Vendor ID
	DriverVersion string  `json:"driver_version"` // Версия драйвера
	Memory        uint64  `json:"memory"`         // Объем памяти (байты)
	MemoryUsed    uint64  `json:"memory_used"`    // Использовано памяти (байты)
	Temperature   float64 `json:"temperature"`    // Температура (°C)
	FanSpeed      uint8   `json:"fan_speed"`      // Скорость вентилятора (%)
	ClockGPU      uint32  `json:"clock_gpu"`      // Частота GPU (MHz)
	ClockMemory   uint32  `json:"clock_memory"`   // Частота памяти (MHz)
	LoadGPU       uint8   `json:"load_gpu"`       // Загрузка GPU (%)
	LoadMemory    uint8   `json:"load_memory"`    // Загрузка памяти (%)
	LoadVideo     uint8   `json:"load_video"`     // Загрузка видео движка (%)
	Resolution    string  `json:"resolution"`     // Разрешение дисплея
	RefreshRate   uint8   `json:"refresh_rate"`   // Частота обновления (Hz)
}

// ToPrint форматирует GPUInfo для вывода
func (g *GPUInfo) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Name", g.Name, "")
	b.AddField("Manufacturer", g.Manufacturer, "")

	if g.Memory > 0 {
		memStr, _ := formatBytes(g.Memory)
		b.AddField("Memory", memStr, "")
	}

	if g.DriverVersion != "" {
		b.AddField("Driver Version", g.DriverVersion, "")
	}

	if g.Temperature > 0 {
		b.AddField("Temperature", fmt.Sprintf("%.1f", g.Temperature), "°C")
	}

	if g.FanSpeed > 0 {
		b.AddField("Fan Speed", fmt.Sprintf("%d", g.FanSpeed), "%")
	}

	if g.ClockGPU > 0 {
		b.AddField("GPU Clock", fmt.Sprintf("%d", g.ClockGPU), "MHz")
	}

	if g.ClockMemory > 0 {
		b.AddField("Memory Clock", fmt.Sprintf("%d", g.ClockMemory), "MHz")
	}

	if g.LoadGPU > 0 || g.LoadMemory > 0 {
		b.AddField("GPU Load", fmt.Sprintf("%d", g.LoadGPU), "%")
		b.AddField("Memory Load", fmt.Sprintf("%d", g.LoadMemory), "%")
	}

	if g.Resolution != "" {
		b.AddField("Resolution", g.Resolution, "")
		if g.RefreshRate > 0 {
			b.AddField("Refresh Rate", fmt.Sprintf("%d", g.RefreshRate), "Hz")
		}
	}

	return b.Build()
}

// GPUInfosToPrint форматирует список GPUInfo
func GPUInfosToPrint(gpus []GPUInfo) string {
	var sb strings.Builder
	for i, gpu := range gpus {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("  GPU %d:\n", i+1))
		sb.WriteString(gpu.ToPrint())
	}
	return sb.String()
}

// GetGPUNameByDeviceID возвращает название GPU по Device ID
func GetGPUNameByDeviceID(deviceID string) string {
	// AMD
	if name, ok := constants.AMDGPUIDs[deviceID]; ok {
		return name
	}
	// Intel
	if name, ok := constants.IntelGPUIDs[deviceID]; ok {
		return name
	}
	// NVIDIA
	if name, ok := constants.NVIDIAGPUIDs[deviceID]; ok {
		return name
	}
	return ""
}
