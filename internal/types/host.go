package types

import (
	"fmt"

	"system-stats/internal/helpers"
)

// HostInfo информация о хосте
type HostInfo struct {
	Hostname        string `json:"hostname"`          // Имя хоста
	Uptime          uint64 `json:"uptime"`            // Время работы (секунды)
	OS              string `json:"os"`                // Операционная система
	Platform        string `json:"platform"`          // Платформа (Windows)
	PlatformFamily  string `json:"platform_family"`   // Семейство платформы
	PlatformVersion string `json:"platform_version"`  // Версия платформы
	KernelVersion   string `json:"kernel_version"`    // Версия ядра
	KernelArch      string `json:"kernel_arch"`       // Архитектура ядра
	Virtualization  string `json:"virtualization"`    // Виртуализация
	Role            string `json:"role"`              // Роль (Workstation/Server)
}

// ToPrint форматирует HostInfo для вывода
func (h *HostInfo) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Hostname", h.Hostname, "")
	b.AddField("OS", h.OS, "")
	b.AddField("Platform", h.Platform, "")
	b.AddField("Platform Version", h.PlatformVersion, "")
	b.AddField("Kernel Version", h.KernelVersion, "")
	b.AddField("Architecture", h.KernelArch, "")
	
	if h.Virtualization != "" {
		b.AddField("Virtualization", h.Virtualization, "")
	}
	if h.Role != "" {
		b.AddField("Role", h.Role, "")
	}

	// Форматируем uptime
	if h.Uptime > 0 {
		uptimeStr := formatUptime(h.Uptime)
		b.AddField("Uptime", uptimeStr, "")
	}

	return b.Build()
}

// formatUptime форматирует uptime в человекочитаемый формат
func formatUptime(seconds uint64) string {
	const (
		minute = 60
		hour   = minute * 60
		day    = hour * 24
	)

	switch {
	case seconds >= day:
		days := seconds / day
		hours := (seconds % day) / hour
		return fmt.Sprintf("%dd %dh", days, hours)
	case seconds >= hour:
		hours := seconds / hour
		minutes := (seconds % hour) / minute
		return fmt.Sprintf("%dh %dm", hours, minutes)
	case seconds >= minute:
		minutes := seconds / minute
		secs := seconds % minute
		return fmt.Sprintf("%dm %ds", minutes, secs)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}

// LoadAvg средняя загрузка системы
type LoadAvg struct {
	Load1  float64 `json:"load1"`   // Загрузка за 1 минуту
	Load5  float64 `json:"load5"`   // Загрузка за 5 минут
	Load15 float64 `json:"load15"`  // Загрузка за 15 минут
}

// ToPrint форматирует LoadAvg для вывода
func (l *LoadAvg) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("1 min", fmt.Sprintf("%.2f", l.Load1), "")
	b.AddField("5 min", fmt.Sprintf("%.2f", l.Load5), "")
	b.AddField("15 min", fmt.Sprintf("%.2f", l.Load15), "")

	return b.Build()
}
