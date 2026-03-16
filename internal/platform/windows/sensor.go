package windows

import (
	"strings"

	"system-stats/internal/constants"
	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// msAcpiThermalZone структура для MSAcpi_ThermalZoneTemperature
type msAcpiThermalZone struct {
	Name             string  `json:"Name"`
	CurrentTemperature uint32 `json:"CurrentTemperature"` // В Kelvin * 10
	HighTemperature    uint32 `json:"HighTemperature"`    // В Kelvin * 10
	CriticalTemperature uint32 `json:"CriticalTemperature"` // В Kelvin * 10
}

// wmiSensor структура для WMI сенсоров
type wmiSensor struct {
	Name        string  `json:"Name"`
	SensorType  string  `json:"SensorType"`
	Value       float64 `json:"Value"`
	Unit        string  `json:"Unit"`
}

// NewSensorTemperatures получает информацию о температурах
func NewSensorTemperatures() ([]types.SensorTemperature, error) {
	result := make([]types.SensorTemperature, 0)

	// Получаем данные из MSAcpi_ThermalZoneTemperature
	script := `
		Get-CimInstance -Namespace ` + constants.RootWMI + ` ` + constants.MSAcpiThermalZoneTemperature + ` 2>$null | `+
			`Select-Object Name,CurrentTemperature,HighTemperature,CriticalTemperature | `+
			`ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return result, nil // Не считаем ошибкой если нет данных
	}

	var thermalZones []msAcpiThermalZone
	if err := helpers.ParseJSON(string(output), &thermalZones); err != nil {
		var single msAcpiThermalZone
		if err2 := helpers.ParseJSON(string(output), &single); err2 == nil {
			thermalZones = []msAcpiThermalZone{single}
		} else {
			return result, nil
		}
	}

	for _, tz := range thermalZones {
		if tz.CurrentTemperature == 0 {
			continue
		}

		// Конвертируем из Kelvin * 10 в Celsius
		tempC := float64(tz.CurrentTemperature)/10.0 - 273.15
		highC := float64(tz.HighTemperature)/10.0 - 273.15
		critC := float64(tz.CriticalTemperature)/10.0 - 273.15

		// Определяем тип сенсора по имени
		sensorType := getSensorType(tz.Name)

		sensor := types.SensorTemperature{
			Name:        tz.Name,
			SensorType:  sensorType,
			Temperature: tempC,
			High:        highC,
			Crit:        critC,
		}

		result = append(result, sensor)
	}

	return result, nil
}

// getSensorType определяет тип сенсора по имени
func getSensorType(name string) string {
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, "cpu") {
		return "CPU"
	}
	if strings.Contains(nameLower, "gpu") || strings.Contains(nameLower, "graphics") {
		return "GPU"
	}
	if strings.Contains(nameLower, "ambient") {
		return "Ambient"
	}
	if strings.Contains(nameLower, "mb") || strings.Contains(nameLower, "motherboard") || strings.Contains(nameLower, "system") {
		return "Motherboard"
	}
	if strings.Contains(nameLower, "battery") {
		return "Battery"
	}

	return "Unknown"
}
