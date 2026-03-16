package types

import (
	"fmt"
	"strings"

	"system-stats/internal/helpers"
)

// VirtualMemory информация об оперативной памяти
type VirtualMemory struct {
	Total       uint64  `json:"total"`        // Общая память (байты)
	Available   uint64  `json:"available"`    // Доступная память (байты)
	Used        uint64  `json:"used"`         // Использовано (байты)
	Free        uint64  `json:"free"`         // Свободно (байты)
	Percent     float64 `json:"percent"`      // Процент использования
	Active      uint64  `json:"active"`       // Активная память (байты)
	Inactive    uint64  `json:"inactive"`     // Неактивная память (байты)
	Cached      uint64  `json:"cached"`       // Кэшировано (байты)
	Buffers     uint64  `json:"buffers"`      // Буферы (байты)
	Shared      uint64  `json:"shared"`       // Разделяемая память (байты)
	Wired       uint64  `json:"wired"`        // Зафиксированная память (байты)
	Committed   uint64  `json:"committed"`    // Закоммиченная память (байты)
	CommitLimit uint64  `json:"commit_limit"` // Лимит коммита (байты)
	PageFile    uint64  `json:"page_file"`    // Файл подкачки (байты)
}

// ToPrint форматирует VirtualMemory для вывода
func (m *VirtualMemory) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddFieldWithFormatter("Total", m.Total, "", formatBytes)
	b.AddFieldWithFormatter("Available", m.Available, "", formatBytes)
	b.AddFieldWithFormatter("Used", m.Used, "", formatBytes)
	b.AddFieldWithFormatter("Free", m.Free, "", formatBytes)
	b.AddField("Percent", fmt.Sprintf("%.2f", m.Percent), "%")

	if m.Active > 0 {
		b.AddFieldWithFormatter("Active", m.Active, "", formatBytes)
	}
	if m.Inactive > 0 {
		b.AddFieldWithFormatter("Inactive", m.Inactive, "", formatBytes)
	}
	if m.Cached > 0 {
		b.AddFieldWithFormatter("Cached", m.Cached, "", formatBytes)
	}
	if m.Buffers > 0 {
		b.AddFieldWithFormatter("Buffers", m.Buffers, "", formatBytes)
	}
	if m.Shared > 0 {
		b.AddFieldWithFormatter("Shared", m.Shared, "", formatBytes)
	}
	if m.Wired > 0 {
		b.AddFieldWithFormatter("Wired", m.Wired, "", formatBytes)
	}
	if m.Committed > 0 {
		b.AddFieldWithFormatter("Committed", m.Committed, "", formatBytes)
	}
	if m.CommitLimit > 0 {
		b.AddFieldWithFormatter("Commit Limit", m.CommitLimit, "", formatBytes)
	}
	if m.PageFile > 0 {
		b.AddFieldWithFormatter("Page File", m.PageFile, "", formatBytes)
	}

	return b.Build()
}

// SwapDevice информация о файле подкачки
type SwapDevice struct {
	Name      string `json:"name"`       // Имя устройства
	Total     uint64 `json:"total"`      // Общий размер (байты)
	Used      uint64 `json:"used"`       // Использовано (байты)
	Free      uint64 `json:"free"`       // Свободно (байты)
	Percent   float64 `json:"percent"`   // Процент использования
	CurrentSize uint64 `json:"current_size"` // Текущий размер (байты)
	PeakSize  uint64 `json:"peak_size"`  // Пиковый размер (байты)
}

// ToPrint форматирует SwapDevice для вывода
func (s *SwapDevice) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Name", s.Name, "")
	b.AddFieldWithFormatter("Total", s.Total, "", formatBytes)
	b.AddFieldWithFormatter("Used", s.Used, "", formatBytes)
	b.AddFieldWithFormatter("Free", s.Free, "", formatBytes)
	b.AddField("Percent", fmt.Sprintf("%.2f", s.Percent), "%")

	if s.CurrentSize > 0 {
		b.AddFieldWithFormatter("Current Size", s.CurrentSize, "", formatBytes)
	}
	if s.PeakSize > 0 {
		b.AddFieldWithFormatter("Peak Size", s.PeakSize, "", formatBytes)
	}

	return b.Build()
}

// SwapDevicesToPrint форматирует список SwapDevice
func SwapDevicesToPrint(devices []SwapDevice) string {
	var sb strings.Builder
	for i, dev := range devices {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("  Swap %d:\n", i+1))
		sb.WriteString(dev.ToPrint())
	}
	return sb.String()
}

// formatBytes форматирует размер в байтах в человекочитаемый формат
func formatBytes(value any) (string, string) {
	var bytes uint64
	switch v := value.(type) {
	case uint64:
		bytes = v
	case int64:
		bytes = uint64(v)
	case uint32:
		bytes = uint64(v)
	case int32:
		bytes = uint64(v)
	case float64:
		bytes = uint64(v)
	default:
		return fmt.Sprintf("%v", value), ""
	}

	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f", float64(bytes)/TB), "TB"
	case bytes >= GB:
		return fmt.Sprintf("%.2f", float64(bytes)/GB), "GB"
	case bytes >= MB:
		return fmt.Sprintf("%.2f", float64(bytes)/MB), "MB"
	case bytes >= KB:
		return fmt.Sprintf("%.2f", float64(bytes)/KB), "KB"
	default:
		return fmt.Sprintf("%d", bytes), "B"
	}
}
