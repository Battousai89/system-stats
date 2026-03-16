package types

import (
	"fmt"
	"strings"

	"system-stats/internal/helpers"
)

// BatteryInfo информация о батарее
type BatteryInfo struct {
	Name             string  `json:"name"`              // Имя батареи
	Manufacturer     string  `json:"manufacturer"`      // Производитель
	SerialNumber     string  `json:"serial_number"`     // Серийный номер
	Chemistry        string  `json:"chemistry"`         // Тип химии (Li-ion, Li-poly, etc.)
	DesignedCapacity uint64  `json:"designed_capacity"` // Проектная емкость (mWh)
	FullChargeCap    uint64  `json:"full_charge_cap"`   // Полная емкость (mWh)
	CurrentCapacity  uint64  `json:"current_capacity"`  // Текущая емкость (mWh)
	Percent          uint8   `json:"percent"`           // Процент заряда
	Status           string  `json:"status"`            // Статус (Charging, Discharging, etc.)
	Voltage          uint32  `json:"voltage"`           // Напряжение (mV)
	EstTimeRemaining uint32  `json:"est_time_remaining"` // Оставшееся время (секунды)
}

// ToPrint форматирует BatteryInfo для вывода
func (b *BatteryInfo) ToPrint() string {
	bld := helpers.NewBuilder()

	bld.AddField("Name", b.Name, "")
	
	if b.Manufacturer != "" {
		bld.AddField("Manufacturer", b.Manufacturer, "")
	}
	if b.SerialNumber != "" {
		bld.AddField("Serial Number", b.SerialNumber, "")
	}
	if b.Chemistry != "" {
		bld.AddField("Chemistry", b.Chemistry, "")
	}

	bld.AddField("Designed Capacity", b.DesignedCapacity, "mWh")
	bld.AddField("Full Charge Capacity", b.FullChargeCap, "mWh")
	bld.AddField("Current Capacity", b.CurrentCapacity, "mWh")
	bld.AddField("Charge Level", fmt.Sprintf("%d", b.Percent), "%")
	bld.AddField("Status", b.Status, "")

	if b.Voltage > 0 {
		bld.AddField("Voltage", fmt.Sprintf("%d", b.Voltage), "mV")
	}
	if b.EstTimeRemaining > 0 {
		timeStr := formatBatteryTime(b.EstTimeRemaining)
		bld.AddField("Time Remaining", timeStr, "")
	}

	return bld.Build()
}

// BatteryInfosToPrint форматирует список BatteryInfo
func BatteryInfosToPrint(batteries []BatteryInfo) string {
	var sb strings.Builder
	for i, bat := range batteries {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("  Battery %d:\n", i+1))
		sb.WriteString(bat.ToPrint())
	}
	return sb.String()
}

// formatBatteryTime форматирует время в секундах в человекочитаемый формат
func formatBatteryTime(seconds uint32) string {
	if seconds == 0 {
		return "Unknown"
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
