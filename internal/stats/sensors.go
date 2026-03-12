package stats

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"system-stats/internal/formatter"
)

type SensorTemperature struct {
	SensorKey   string  `json:"sensorKey"`
	Temperature float64 `json:"temperature"`
	High        float64 `json:"high"`
	Critical    float64 `json:"critical"`
}

func NewSensorTemperatures() ([]SensorTemperature, error) {
	switch runtime.GOOS {
	case "linux":
		return parseHwmonSensors()
	case "windows":
		return getWindowsSensors()
	case "darwin":
		return getDarwinSensors()
	default:
		return []SensorTemperature{}, nil
	}
}

func parseHwmonSensors() ([]SensorTemperature, error) {
	var result []SensorTemperature

	entries, err := os.ReadDir("/sys/class/hwmon")
	if err != nil {
		return result, nil
	}

	for _, entry := range entries {
		name := entry.Name()
		sensorName := getHwmonName(name)

		tempFiles, err := os.ReadDir("/sys/class/hwmon/" + name)
		if err != nil {
			continue
		}

		for _, f := range tempFiles {
			if strings.HasPrefix(f.Name(), "temp") && strings.HasSuffix(f.Name(), "_input") {
				tempPath := "/sys/class/hwmon/" + name + "/" + f.Name()
				temp, err := readTempFile(tempPath)
				if err != nil || temp == 0 {
					continue
				}

				temperature := float64(temp) / 1000.0

				baseName := strings.TrimSuffix(f.Name(), "_input")
				high, _ := readTempFile("/sys/class/hwmon/" + name + "/" + baseName + "_max")
				critical, _ := readTempFile("/sys/class/hwmon/" + name + "/" + baseName + "_crit")

				sensorKey := sensorName
				if !strings.Contains(f.Name(), "1") {
					sensorKey += "_" + strings.TrimSuffix(f.Name(), "_input")
				}

				result = append(result, SensorTemperature{
					SensorKey:   sensorKey,
					Temperature: temperature,
					High:        float64(high) / 1000.0,
					Critical:    float64(critical) / 1000.0,
				})
			}
		}
	}

	thermalEntries, err := os.ReadDir("/sys/class/thermal")
	if err == nil {
		for _, entry := range thermalEntries {
			if strings.HasPrefix(entry.Name(), "thermal_zone") {
				tempPath := "/sys/class/thermal/" + entry.Name() + "/temp"
				temp, err := readTempFile(tempPath)
				if err != nil || temp == 0 {
					continue
				}

				typeData, _ := os.ReadFile("/sys/class/thermal/" + entry.Name() + "/type")
				sensorType := strings.TrimSpace(string(typeData))
				if sensorType == "" {
					sensorType = entry.Name()
				}

				result = append(result, SensorTemperature{
					SensorKey:   sensorType,
					Temperature: float64(temp) / 1000.0,
					High:        0,
					Critical:    0,
				})
			}
		}
	}

	return result, nil
}

func getHwmonName(hwmon string) string {
	namePath := "/sys/class/hwmon/" + hwmon + "/name"
	data, err := os.ReadFile(namePath)
	if err == nil {
		return strings.TrimSpace(string(data))
	}

	devicePath := "/sys/class/hwmon/" + hwmon + "/device/name"
	data, err = os.ReadFile(devicePath)
	if err == nil {
		return strings.TrimSpace(string(data))
	}

	return hwmon
}

func readTempFile(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	temp, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}

	return temp, nil
}

func getWindowsSensors() ([]SensorTemperature, error) {
	var result []SensorTemperature

	output, err := runCommandWithTimeout("wmic", "/namespace:\\\\root\\wmi", "PATH", "MSAcpi_ThermalZoneTemperature", "get", "CurrentTemperature", "/format:csv")
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue
			}
			temp, _ := strconv.ParseInt(strings.TrimSpace(line), 10, 64)
			if temp > 0 {
				celsius := float64(temp)/10.0 - 273.15
				result = append(result, SensorTemperature{
					SensorKey:   "CPU",
					Temperature: celsius,
				})
			}
		}
	}

	return result, nil
}

func getDarwinSensors() ([]SensorTemperature, error) {
	var result []SensorTemperature

	output, err := runCommandWithTimeout("sudo", "powermetrics", "--samplers", "smc", "-n", "1", "-i", "100")
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "CPU die temperature") {
				var temp float64
				fmt.Sscanf(line, "%*s %*s %*s %f", &temp)
				result = append(result, SensorTemperature{
					SensorKey:   "CPU",
					Temperature: temp,
				})
			} else if strings.Contains(line, "GPU die temperature") {
				var temp float64
				fmt.Sscanf(line, "%*s %*s %*s %f", &temp)
				result = append(result, SensorTemperature{
					SensorKey:   "GPU",
					Temperature: temp,
				})
			}
		}
	}

	return result, nil
}

func (t SensorTemperature) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Sensor", t.SensorKey, "").
		AddField("Temperature", t.Temperature, "°C").
		AddField("High", t.High, "°C").
		AddField("Critical", t.Critical, "°C").
		Build()
}

func SensorTemperaturesToPrint(temps []SensorTemperature) string {
	var sb strings.Builder
	for i, t := range temps {
		sb.WriteString(t.ToPrint())
		if i < len(temps)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
