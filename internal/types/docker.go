package types

import (
	"fmt"
	"strings"

	"system-stats/internal/helpers"
)

// DockerStats статистика Docker контейнера
type DockerStats struct {
	ContainerID   string  `json:"container_id"`   // ID контейнера
	Name          string  `json:"name"`           // Имя контейнера
	CPU           float64 `json:"cpu"`            // Использование CPU (%)
	Memory        uint64  `json:"memory"`         // Использование памяти (байты)
	MemoryLimit   uint64  `json:"memory_limit"`   // Лимит памяти (байты)
	MemoryPercent float64 `json:"memory_percent"` // Процент памяти
	NetIO         string  `json:"net_io"`         // Сетевой I/O
	BlockIO       string  `json:"block_io"`       // Блочный I/O
	PIDs          uint32  `json:"pids"`           // Количество процессов
	Status        string  `json:"status"`         // Статус контейнера
}

// ToPrint форматирует DockerStats для вывода
func (d *DockerStats) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Name", d.Name, "")
	b.AddField("ID", d.ContainerID[:12], "")
	b.AddField("CPU", fmt.Sprintf("%.2f", d.CPU), "%")

	memStr, _ := formatBytes(d.Memory)
	b.AddField("Memory", memStr, "")
	b.AddField("Memory %", fmt.Sprintf("%.1f", d.MemoryPercent), "%")
	b.AddField("PIDs", d.PIDs, "")
	b.AddField("Status", d.Status, "")

	return b.Build()
}

// DockerStatsToPrint форматирует список DockerStats
func DockerStatsToPrint(stats []DockerStats) string {
	var sb strings.Builder

	if len(stats) == 0 {
		sb.WriteString("  No running containers\n")
		return sb.String()
	}

	// Заголовок таблицы
	sb.WriteString(fmt.Sprintf("  %-15s %-25s %8s %10s %6s %s\n",
		"Container ID", "Name", "CPU%", "Memory", "PIDs", "Status"))
	sb.WriteString("  " + strings.Repeat("─", 80) + "\n")

	for _, s := range stats {
		memStr, _ := formatBytes(s.Memory)
		id := s.ContainerID
		if len(id) > 12 {
			id = id[:12]
		}
		sb.WriteString(fmt.Sprintf("  %-15s %-25s %7.2f%% %9s %6d %s\n",
			id, truncateString(s.Name, 25), s.CPU, memStr, s.PIDs, s.Status))
	}

	return sb.String()
}
