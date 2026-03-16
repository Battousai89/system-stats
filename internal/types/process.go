package types

import (
	"fmt"
	"strings"

	"system-stats/internal/helpers"
)

// ProcessInfo информация о процессе
type ProcessInfo struct {
	PID         uint32  `json:"pid"`          // ID процесса
	Name        string  `json:"name"`         // Имя процесса
	CPU         float64 `json:"cpu"`          // Использование CPU (%)
	Memory      uint64  `json:"memory"`       // Использование памяти (байты)
	MemoryPercent float64 `json:"memory_percent"` // Процент памяти
	Status      string  `json:"status"`       // Статус процесса
	Username    string  `json:"username"`     // Пользователь
	Cmdline     string  `json:"cmdline"`      // Командная строка
	CreateTime  uint64  `json:"create_time"`  // Время создания (unix timestamp)
	NumThreads  uint32  `json:"num_threads"`  // Количество потоков
}

// ToPrint форматирует ProcessInfo для вывода
func (p *ProcessInfo) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("PID", p.PID, "")
	b.AddField("Name", p.Name, "")
	b.AddField("CPU", fmt.Sprintf("%.1f", p.CPU), "%")
	
	memStr, _ := formatBytes(p.Memory)
	b.AddField("Memory", memStr, "")
	b.AddField("Memory %", fmt.Sprintf("%.1f", p.MemoryPercent), "%")
	
	if p.Username != "" {
		b.AddField("User", p.Username, "")
	}
	if p.NumThreads > 0 {
		b.AddField("Threads", p.NumThreads, "")
	}
	if p.Status != "" {
		b.AddField("Status", p.Status, "")
	}

	return b.Build()
}

// ProcessInfosToPrint форматирует список ProcessInfo
func ProcessInfosToPrint(processes []ProcessInfo) string {
	var sb strings.Builder
	
	// Заголовок таблицы
	sb.WriteString(fmt.Sprintf("  %-8s %-25s %8s %10s %8s %s\n",
		"PID", "Name", "CPU%", "Memory", "Threads", "User"))
	sb.WriteString("  " + strings.Repeat("─", 80) + "\n")

	for _, p := range processes {
		memStr, _ := formatBytes(p.Memory)
		user := p.Username
		if user == "" {
			user = "-"
		}
		sb.WriteString(fmt.Sprintf("  %-8d %-25s %7.1f%% %9s %8d %s\n",
			p.PID, truncateString(p.Name, 25), p.CPU, memStr, p.NumThreads, user))
	}

	return sb.String()
}

// truncateString обрезает строку до максимальной длины
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
