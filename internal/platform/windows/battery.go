package windows

import (
	"fmt"
	"strings"

	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// win32Battery структура для Win32_Battery
type win32Battery struct {
	Name             string `json:"Name"`
	Manufacturer     string `json:"Manufacturer"`
	SerialNumber     string `json:"SerialNumber"`
	Chemistry        string `json:"Chemistry"`
	DesignCapacity   uint64 `json:"DesignCapacity"`
	FullChargeCapacity uint64 `json:"FullChargeCapacity"`
	EstimatedChargeRemaining uint32 `json:"EstimatedChargeRemaining"`
	EstimatedRunTime uint32 `json:"EstimatedRunTime"`
	ExpectedLife     uint32 `json:"ExpectedLife"`
	MaxRechargeTime  uint32 `json:"MaxRechargeTime"`
	Status           string `json:"Status"`
	BatteryStatus    uint16 `json:"BatteryStatus"`
}

// NewBatteryInfo получает информацию о батареях
func NewBatteryInfo() ([]types.BatteryInfo, error) {
	script := `
		Get-CimInstance Win32_Battery | `+
			`Select-Object Name,Manufacturer,SerialNumber,Chemistry,`+
			`DesignCapacity,FullChargeCapacity,EstimatedChargeRemaining,`+
			`EstimatedRunTime,ExpectedLife,MaxRechargeTime,Status,BatteryStatus | `+
			`ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get battery info: %w", err)
	}

	var batteries []win32Battery
	if err := helpers.ParseJSON(string(output), &batteries); err != nil {
		var single win32Battery
		if err2 := helpers.ParseJSON(string(output), &single); err2 == nil {
			batteries = []win32Battery{single}
		} else {
			return nil, nil // Нет батарей - не ошибка
		}
	}

	result := make([]types.BatteryInfo, 0, len(batteries))
	for _, b := range batteries {
		// Пропускаем если нет данных о емкости
		if b.DesignCapacity == 0 && b.FullChargeCapacity == 0 {
			continue
		}

		// Конвертируем DesignCapacity из Wh в mWh (если < 1000, значит в Wh)
		designedCap := b.DesignCapacity
		if designedCap < 1000 && designedCap > 0 {
			designedCap *= 1000
		}

		fullCap := b.FullChargeCapacity
		if fullCap < 1000 && fullCap > 0 {
			fullCap *= 1000
		}

		// Текущая емкость в mWh
		currentCap := uint64(b.EstimatedChargeRemaining) * fullCap / 100

		battery := types.BatteryInfo{
			Name:             b.Name,
			Manufacturer:     b.Manufacturer,
			SerialNumber:     strings.TrimSpace(b.SerialNumber),
			Chemistry:        b.Chemistry,
			DesignedCapacity: designedCap,
			FullChargeCap:    fullCap,
			CurrentCapacity:  currentCap,
			Percent:          uint8(b.EstimatedChargeRemaining),
			Status:           getBatteryStatus(b.BatteryStatus),
			EstTimeRemaining: b.EstimatedRunTime * 60, // Конвертируем минуты в секунды
		}

		result = append(result, battery)
	}

	return result, nil
}

// getBatteryStatus конвертирует BatteryStatus в строку
func getBatteryStatus(status uint16) string {
	switch status {
	case 1:
		return "Other"
	case 2:
		return "Unknown"
	case 3:
		return "Fully Charged"
	case 4:
		return "Low"
	case 5:
		return "Critical"
	case 6:
		return "Charging"
	case 7:
		return "Charging and High"
	case 8:
		return "Charging and Low"
	case 9:
		return "Charging and Critical"
	case 10:
		return "Undefined"
	case 11:
		return "Partially Charged"
	default:
		return "Unknown"
	}
}
