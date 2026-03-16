//go:build linux
// +build linux

package linux

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"system-stats/internal/types"
)

// NewSensorTemperatures gets hardware sensor temperatures on Linux
func NewSensorTemperatures() ([]types.SensorTemperature, error) {
	var result []types.SensorTemperature

	// Read from hwmon (hardware monitoring)
	if sensors, err := readHwmonSensors(); err == nil && len(sensors) > 0 {
		result = append(result, sensors...)
	}

	// Read from thermal zones
	if sensors, err := readThermalZoneSensors(); err == nil && len(sensors) > 0 {
		result = append(result, sensors...)
	}

	return result, nil
}

// readHwmonSensors reads temperatures from hwmon subsystem
func readHwmonSensors() ([]types.SensorTemperature, error) {
	var result []types.SensorTemperature

	// Find all hwmon devices
	matches, err := filepath.Glob("/sys/class/hwmon/hwmon*")
	if err != nil {
		return nil, err
	}

	for _, hwmonPath := range matches {
		sensors := parseHwmonDevice(hwmonPath)
		result = append(result, sensors...)
	}

	return result, nil
}

// parseHwmonDevice parses a hwmon device
func parseHwmonDevice(hwmonPath string) []types.SensorTemperature {
	var result []types.SensorTemperature

	// Get device name
	name, _ := os.ReadFile(filepath.Join(hwmonPath, "name"))
	deviceName := strings.TrimSpace(string(name))
	if deviceName == "" {
		deviceName = filepath.Base(hwmonPath)
	}

	// Read all temperature inputs
	tempFiles, _ := filepath.Glob(filepath.Join(hwmonPath, "temp*_input"))
	
	for _, tempFile := range tempFiles {
		content, err := os.ReadFile(tempFile)
		if err != nil {
			continue
		}

		temp, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
		if err != nil {
			continue
		}

		// Temperature is in millidegrees Celsius
		tempC := temp / 1000.0

		// Sanity check
		if tempC < -50 || tempC > 150 {
			continue
		}

		// Extract sensor number from filename
		sensorNum := extractSensorNumber(tempFile)

		// Try to get sensor label
		labelPath := filepath.Join(hwmonPath, "temp"+sensorNum+"_label")
		label, _ := os.ReadFile(labelPath)
		sensorName := strings.TrimSpace(string(label))
		if sensorName == "" {
			sensorName = deviceName + " " + sensorNum
		}

		// Try to get high and critical thresholds
		high, _ := readThreshold(hwmonPath, sensorNum, "max")
		crit, _ := readThreshold(hwmonPath, sensorNum, "crit")

		sensor := types.SensorTemperature{
			Name:        sensorName,
			SensorType:  getSensorType(deviceName, sensorName),
			Temperature: tempC,
			High:        high,
			Crit:        crit,
		}

		result = append(result, sensor)
	}

	return result
}

// readThermalZoneSensors reads temperatures from thermal zone subsystem
func readThermalZoneSensors() ([]types.SensorTemperature, error) {
	var result []types.SensorTemperature

	// Find all thermal zones
	matches, err := filepath.Glob("/sys/class/thermal/thermal_zone*")
	if err != nil {
		return nil, err
	}

	for _, zonePath := range matches {
		// Read temperature
		tempPath := filepath.Join(zonePath, "temp")
		content, err := os.ReadFile(tempPath)
		if err != nil {
			continue
		}

		temp, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
		if err != nil {
			continue
		}

		// Temperature is in millidegrees Celsius
		tempC := temp / 1000.0

		// Sanity check
		if tempC < -50 || tempC > 150 {
			continue
		}

		// Read zone name
		namePath := filepath.Join(zonePath, "type")
		name, _ := os.ReadFile(namePath)
		zoneName := strings.TrimSpace(string(name))
		if zoneName == "" {
			zoneName = filepath.Base(zonePath)
		}

		// Read trip points (thresholds)
		high, crit := readThermalTripPoints(zonePath)

		sensor := types.SensorTemperature{
			Name:        zoneName,
			SensorType:  getSensorType("thermal", zoneName),
			Temperature: tempC,
			High:        high,
			Crit:        crit,
		}

		result = append(result, sensor)
	}

	return result, nil
}

// readThermalTripPoints reads trip point temperatures
func readThermalTripPoints(zonePath string) (high, crit float64) {
	// Try to read trip points
	for i := 0; i < 10; i++ {
		tripPath := filepath.Join(zonePath, "trip_point_"+strconv.Itoa(i)+"_temp")
		content, err := os.ReadFile(tripPath)
		if err != nil {
			continue
		}

		temp, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
		if err != nil {
			continue
		}

		tempC := temp / 1000.0

		// First trip point is usually passive/high, second is critical
		if high == 0 {
			high = tempC
		} else if crit == 0 {
			crit = tempC
			break
		}
	}

	return high, crit
}

// readThreshold reads temperature threshold
func readThreshold(hwmonPath, sensorNum, thresholdType string) (float64, error) {
	path := filepath.Join(hwmonPath, "temp"+sensorNum+"_"+thresholdType)
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	temp, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
	if err != nil {
		return 0, err
	}

	return temp / 1000.0, nil
}

// extractSensorNumber extracts sensor number from filename
func extractSensorNumber(filename string) string {
	base := filepath.Base(filename)
	// Format: tempN_input
	parts := strings.Split(base, "_")
	if len(parts) > 0 {
		num := strings.TrimPrefix(parts[0], "temp")
		return num
	}
	return "1"
}

// getSensorType determines sensor type from name
func getSensorType(deviceName, sensorName string) string {
	deviceName = strings.ToLower(deviceName)
	sensorName = strings.ToLower(sensorName)

	// Check for CPU
	if strings.Contains(deviceName, "coretemp") ||
		strings.Contains(deviceName, "k10temp") ||
		strings.Contains(deviceName, "k8temp") ||
		strings.Contains(sensorName, "cpu") ||
		strings.Contains(sensorName, "core") ||
		strings.Contains(sensorName, "tdie") ||
		strings.Contains(sensorName, "tccd") {
		return "CPU"
	}

	// Check for GPU
	if strings.Contains(deviceName, "amdgpu") ||
		strings.Contains(deviceName, "nvidia") ||
		strings.Contains(deviceName, "i915") ||
		strings.Contains(sensorName, "gpu") ||
		strings.Contains(sensorName, "gfx") {
		return "GPU"
	}

	// Check for motherboard/chipset
	if strings.Contains(deviceName, "nct") ||
		strings.Contains(deviceName, "it87") ||
		strings.Contains(deviceName, "f718") ||
		strings.Contains(deviceName, "w83") ||
		strings.Contains(sensorName, "mb") ||
		strings.Contains(sensorName, "board") ||
		strings.Contains(sensorName, "system") ||
		strings.Contains(sensorName, "pch") {
		return "Motherboard"
	}

	// Check for disk drive
	if strings.Contains(deviceName, "drivetemp") ||
		strings.Contains(sensorName, "drive") ||
		strings.Contains(sensorName, "disk") ||
		strings.Contains(sensorName, "hdd") ||
		strings.Contains(sensorName, "ssd") {
		return "Drive"
	}

	// Check for battery
	if strings.Contains(sensorName, "battery") ||
		strings.Contains(sensorName, "bat") {
		return "Battery"
	}

	// Check for ambient
	if strings.Contains(sensorName, "ambient") ||
		strings.Contains(sensorName, "fan") {
		return "Ambient"
	}

	return "Unknown"
}
