package types

import (
	"fmt"
	"strings"

	"system-stats/internal/helpers"
)

// SensorTemperature информация о датчике температуры
type SensorTemperature struct {
	Name        string  `json:"name"`         // Имя сенсора
	SensorType  string  `json:"sensor_type"`  // Тип сенсора (CPU, GPU, Motherboard, etc.)
	Temperature float64 `json:"temperature"`  // Температура (°C)
	High        float64 `json:"high"`         // Максимальная температура (°C)
	Crit        float64 `json:"crit"`         // Критическая температура (°C)
}

// ToPrint форматирует SensorTemperature для вывода
func (s *SensorTemperature) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Name", s.Name, "")
	b.AddField("Type", s.SensorType, "")
	b.AddField("Temperature", fmt.Sprintf("%.1f", s.Temperature), "°C")
	
	if s.High > 0 {
		b.AddField("High", fmt.Sprintf("%.1f", s.High), "°C")
	}
	if s.Crit > 0 {
		b.AddField("Critical", fmt.Sprintf("%.1f", s.Crit), "°C")
	}

	return b.Build()
}

// SensorTemperaturesToPrint форматирует список SensorTemperature
func SensorTemperaturesToPrint(sensors []SensorTemperature) string {
	var sb strings.Builder
	for i, sensor := range sensors {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("  Sensor %d:\n", i+1))
		sb.WriteString(sensor.ToPrint())
	}
	return sb.String()
}
