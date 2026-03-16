package types

import (
	"fmt"
	"strings"

	"system-stats/internal/helpers"
)

// CPUInfo информация о процессоре
type CPUInfo struct {
	Name                      string `json:"name"`                         // Модель процессора
	Manufacturer              string `json:"manufacturer"`                 // Производитель
	Family                    string `json:"family"`                       // Семейство
	Model                     string `json:"model"`                        // Модель
	Stepping                  string `json:"stepping"`                     // Степпинг
	Architecture              string `json:"architecture"`                 // Архитектура
	Socket                    string `json:"socket"`                       // Сокет
	L2CacheSize               uint32 `json:"l2_cache_size,omitempty"`      // Размер L2 кэша (KB)
	L3CacheSize               uint32 `json:"l3_cache_size,omitempty"`      // Размер L3 кэша (KB)
	Cores                     uint32 `json:"cores"`                        // Количество ядер
	LogicalProcessors         uint32 `json:"logical_processors"`           // Количество логических процессоров (потоков)
	CurrentClockSpeed         uint64 `json:"current_clock_speed"`          // Текущая частота (MHz)
	MaxClockSpeed             uint64 `json:"max_clock_speed"`              // Максимальная частота (MHz)
	Voltage                   string `json:"voltage,omitempty"`            // Напряжение
	Temperature               uint32 `json:"temperature,omitempty"`        // Температура (°C)
	LoadPercentage            uint8  `json:"load_percentage"`              // Загрузка (%)
	ProcessorType             string `json:"processor_type"`               // Тип процессора
	Status                    string `json:"status"`                       // Статус
	Enabled                   bool   `json:"enabled"`                      // Включен
	Caption                   string `json:"caption"`                      // Описание
	DeviceID                  string `json:"device_id"`                    // ID устройства
	NumberOfCores             uint32 `json:"number_of_cores"`              // Количество ядер (альтернативное поле)
	NumberOfLogicalProcessors uint32 `json:"number_of_logical_processors"` // Количество логических процессоров (альтернативное поле)
}

// CPUTimes времена процессора
type CPUTimes struct {
	CPU       string  `json:"cpu"`       // Имя CPU (CPU Total или номер ядра)
	User      float64 `json:"user"`      // Пользовательское время (сек)
	System    float64 `json:"system"`    // Системное время (сек)
	Idle      float64 `json:"idle"`      // Время простоя (сек)
	Interrupt float64 `json:"interrupt"` // Время обработки прерываний (сек)
	DPC       float64 `json:"dpc"`       // Время отложенных вызовов процедур (сек)
	Total     float64 `json:"total"`     // Общее время (сек)
	Usage     float64 `json:"usage"`     // Процент использования
}

// CPUPercent процент использования процессора
type CPUPercent struct {
	CPU           string  `json:"cpu"`            // Имя CPU (CPU Total или номер ядра)
	Percent       float64 `json:"percent"`        // Процент использования
	UserPercent   float64 `json:"user_percent"`   // Пользовательский процент
	SystemPercent float64 `json:"system_percent"` // Системный процент
	IdlePercent   float64 `json:"idle_percent"`   // Процент простоя
}

// ToPrint форматирует CPUInfo для вывода
func (c *CPUInfo) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Name", c.Name, "")
	b.AddField("Manufacturer", c.Manufacturer, "")
	b.AddField("Family", c.Family, "")
	b.AddField("Model", c.Model, "")
	b.AddField("Stepping", c.Stepping, "")
	b.AddField("Architecture", c.Architecture, "")
	b.AddField("Socket", c.Socket, "")

	if c.L2CacheSize > 0 {
		b.AddField("L2 Cache", c.L2CacheSize, "KB")
	}
	if c.L3CacheSize > 0 {
		b.AddField("L3 Cache", c.L3CacheSize, "KB")
	}

	b.AddField("Cores", c.Cores, "")
	b.AddField("Logical Processors", c.LogicalProcessors, "")
	b.AddField("Current Clock", c.CurrentClockSpeed, "MHz")
	b.AddField("Max Clock", c.MaxClockSpeed, "MHz")

	if c.Voltage != "" {
		b.AddField("Voltage", c.Voltage, "V")
	}
	if c.Temperature > 0 {
		b.AddField("Temperature", c.Temperature, "°C")
	}

	b.AddField("Load", c.LoadPercentage, "%")
	b.AddField("Type", c.ProcessorType, "")
	b.AddField("Status", c.Status, "")

	return b.Build()
}

// ToPrint форматирует CPUTimes для вывода
func (c *CPUTimes) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("CPU", c.CPU, "")
	b.AddFieldWithFormatter("User", c.User, "s", formatSeconds)
	b.AddFieldWithFormatter("System", c.System, "s", formatSeconds)
	b.AddFieldWithFormatter("Idle", c.Idle, "s", formatSeconds)
	b.AddFieldWithFormatter("Interrupt", c.Interrupt, "s", formatSeconds)
	b.AddFieldWithFormatter("DPC", c.DPC, "s", formatSeconds)
	b.AddFieldWithFormatter("Total", c.Total, "s", formatSeconds)
	b.AddField("Usage", fmt.Sprintf("%.2f", c.Usage), "%")

	return b.Build()
}

// ToPrint форматирует CPUPercent для вывода
func (c *CPUPercent) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("CPU", c.CPU, "")
	b.AddField("Total", fmt.Sprintf("%.2f", c.Percent), "%")
	b.AddField("User", fmt.Sprintf("%.2f", c.UserPercent), "%")
	b.AddField("System", fmt.Sprintf("%.2f", c.SystemPercent), "%")
	b.AddField("Idle", fmt.Sprintf("%.2f", c.IdlePercent), "%")

	return b.Build()
}

// CPUInfosToPrint форматирует список CPUInfo
func CPUInfosToPrint(cpus []CPUInfo) string {
	var sb strings.Builder
	for i, cpu := range cpus {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("  CPU %d:\n", i+1))
		sb.WriteString(cpu.ToPrint())
	}
	return sb.String()
}

// CPUTimesToPrint форматирует список CPUTimes
func CPUTimesToPrint(times []CPUTimes) string {
	var sb strings.Builder
	for _, t := range times {
		sb.WriteString(t.ToPrint())
	}
	return sb.String()
}

// CPUPercentsToPrint форматирует список CPUPercent
func CPUPercentsToPrint(percents []CPUPercent) string {
	var sb strings.Builder
	for _, p := range percents {
		sb.WriteString(p.ToPrint())
	}
	return sb.String()
}

func formatSeconds(value any) (string, string) {
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("%.4f", v), "s"
	case float32:
		return fmt.Sprintf("%.4f", v), "s"
	default:
		return fmt.Sprintf("%v", v), "s"
	}
}
