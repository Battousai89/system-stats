package types

import (
	"fmt"

	"system-stats/internal/helpers"
)

// LoadMisc разная информация о загрузке системы
type LoadMisc struct {
	ProcsTotal    uint64  `json:"procs_total"`     // Всего процессов
	ProcsRunning  uint64  `json:"procs_running"`   // Запущенные процессы
	ProcsBlocked  uint64  `json:"procs_blocked"`   // Заблокированные процессы
	ProcsStopped  uint64  `json:"procs_stopped"`   // Остановленные процессы
	ProcsZombie   uint64  `json:"procs_zombie"`    // Zombie процессы
	Uptime        uint64  `json:"uptime"`          // Время работы (секунды)
	UptimeDays    float64 `json:"uptime_days"`     // Время работы (дни)
	BootTime      uint64  `json:"boot_time"`       // Время загрузки (unix timestamp)
	Load1         float64 `json:"load1"`           // Загрузка за 1 минуту
	Load5         float64 `json:"load5"`           // Загрузка за 5 минут
	Load15        float64 `json:"load15"`          // Загрузка за 15 минут
	ContextSwitches uint64 `json:"context_switches"` // Переключения контекста
	Interrupts    uint64  `json:"interrupts"`      // Прерывания
}

// ToPrint форматирует LoadMisc для вывода
func (l *LoadMisc) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Total Processes", l.ProcsTotal, "")
	b.AddField("Running", l.ProcsRunning, "")
	b.AddField("Blocked", l.ProcsBlocked, "")
	b.AddField("Stopped", l.ProcsStopped, "")
	b.AddField("Zombie", l.ProcsZombie, "")

	if l.Uptime > 0 {
		uptimeStr := formatUptimeFromHost(l.Uptime)
		b.AddField("Uptime", uptimeStr, "")
	}

	if l.BootTime > 0 {
		b.AddField("Boot Time", fmt.Sprintf("%d", l.BootTime), "")
	}

	if l.Load1 > 0 || l.Load5 > 0 || l.Load15 > 0 {
		b.AddField("Load Average", fmt.Sprintf("%.2f, %.2f, %.2f", l.Load1, l.Load5, l.Load15), "")
	}

	if l.ContextSwitches > 0 {
		b.AddField("Context Switches", l.ContextSwitches, "")
	}

	if l.Interrupts > 0 {
		b.AddField("Interrupts", l.Interrupts, "")
	}

	return b.Build()
}

// NewLoadMisc создает новую LoadMisc
func NewLoadMisc() *LoadMisc {
	return &LoadMisc{}
}

// formatUptimeFromHost форматирует uptime (копия из host.go для избежания дублирования)
func formatUptimeFromHost(seconds uint64) string {
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
