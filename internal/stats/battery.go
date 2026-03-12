package stats

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"system-stats/internal/formatter"
)

type BatteryInfo struct {
	FullmWh     float64 `json:"fullmWh"`
	DesignmWh   float64 `json:"designmWh"`
	CurrentmWh  float64 `json:"currentmWh"`
	ChargePercent float64 `json:"chargePercent"`
	State       string  `json:"state"`
}

func NewBatteryInfo() ([]BatteryInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return parseLinuxBattery()
	case "windows":
		return getWindowsBattery()
	case "darwin":
		return getDarwinBattery()
	default:
		return []BatteryInfo{}, nil
	}
}

func parseLinuxBattery() ([]BatteryInfo, error) {
	var result []BatteryInfo

	entries, err := os.ReadDir("/sys/class/power_supply")
	if err != nil {
		return result, nil
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "BAT") && !strings.HasPrefix(name, "CMB") {
			typeData, err := os.ReadFile("/sys/class/power_supply/" + name + "/type")
			if err != nil || strings.TrimSpace(string(typeData)) != "Battery" {
				continue
			}
		}

		battery := BatteryInfo{}
		hasEnergyData := false
		hasChargeData := false

		if data, err := os.ReadFile("/sys/class/power_supply/" + name + "/energy_full"); err == nil {
			energy, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
			if energy > 0 {
				battery.FullmWh = float64(energy) / 1000.0
				hasEnergyData = true
			}
		}

		if data, err := os.ReadFile("/sys/class/power_supply/" + name + "/energy_full_design"); err == nil {
			energy, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
			if energy > 0 {
				battery.DesignmWh = float64(energy) / 1000.0
			}
		}

		if data, err := os.ReadFile("/sys/class/power_supply/" + name + "/energy_now"); err == nil {
			energy, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
			if energy > 0 {
				battery.CurrentmWh = float64(energy) / 1000.0
				hasEnergyData = true
			}
		}

		if battery.FullmWh == 0 {
			if data, err := os.ReadFile("/sys/class/power_supply/" + name + "/charge_full"); err == nil {
				charge, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
				if charge > 0 {
					battery.FullmWh = float64(charge) / 1000.0
					hasChargeData = true
				}
			}
		}
		if battery.DesignmWh == 0 {
			if data, err := os.ReadFile("/sys/class/power_supply/" + name + "/charge_full_design"); err == nil {
				charge, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
				if charge > 0 {
					battery.DesignmWh = float64(charge) / 1000.0
				}
			}
		}
		if battery.CurrentmWh == 0 {
			if data, err := os.ReadFile("/sys/class/power_supply/" + name + "/charge_now"); err == nil {
				charge, _ := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
				if charge > 0 {
					battery.CurrentmWh = float64(charge) / 1000.0
					hasChargeData = true
				}
			}
		}

		if data, err := os.ReadFile("/sys/class/power_supply/" + name + "/status"); err == nil {
			battery.State = strings.TrimSpace(string(data))
		}

		if battery.FullmWh > 0 {
			battery.ChargePercent = battery.CurrentmWh / battery.FullmWh * 100
		}

		if battery.DesignmWh > 0 && battery.FullmWh > 0 {
			if battery.DesignmWh < battery.FullmWh*0.5 {
				battery.DesignmWh = battery.FullmWh
			}
		}

		if !hasEnergyData && !hasChargeData {
			continue
		}

		if battery.FullmWh > 0 || battery.CurrentmWh > 0 {
			result = append(result, battery)
		}
	}

	return result, nil
}

func getWindowsBattery() ([]BatteryInfo, error) {
	var result []BatteryInfo

	output, err := runCommandWithTimeout("wmic", "path", "Win32_Battery", "get", "FullChargeCapacity,DesignCapacity,EstimatedChargeRemaining,BatteryStatus,EstimatedRuntime", "/format:csv")
	if err != nil {
		return result, nil
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			continue
		}

		battery := BatteryInfo{}

		full, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		design, _ := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		current, _ := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
		status, _ := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)

		battery.FullmWh = float64(full)
		battery.DesignmWh = float64(design)
		battery.CurrentmWh = float64(current)
		if full > 0 {
			battery.ChargePercent = float64(current) * 100 / float64(full)
		}

		switch status {
		case 1:
			battery.State = "Discharging"
		case 2:
			battery.State = "Charging"
		case 3:
			battery.State = "Idle"
		default:
			battery.State = "Unknown"
		}

		result = append(result, battery)
	}

	return result, nil
}

func getDarwinBattery() ([]BatteryInfo, error) {
	var result []BatteryInfo

	output, err := runCommandWithTimeout("pmset", "-g", "batt")
	if err != nil {
		return result, nil
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "InternalBattery") {
			battery := BatteryInfo{}

			if strings.Contains(line, "charging") {
				battery.State = "Charging"
			} else if strings.Contains(line, "discharging") {
				battery.State = "Discharging"
			} else {
				battery.State = "Idle"
			}

			for _, part := range strings.Split(line, ";") {
				part = strings.TrimSpace(part)
				if strings.Contains(part, "%") {
					pct := strings.TrimSuffix(part, "%")
					percent, _ := strconv.ParseFloat(pct, 64)
					battery.ChargePercent = percent

					maxOutput, err := runCommandWithTimeout("ioreg", "-rc", "AppleSmartBattery")
					if err == nil {
						for _, l := range strings.Split(string(maxOutput), "\n") {
							if strings.Contains(l, "\"MaxCapacity\"") {
								parts := strings.Split(l, "=")
								if len(parts) > 1 {
									max, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
									battery.FullmWh = max
									battery.CurrentmWh = max * percent / 100
									battery.DesignmWh = max
								}
							}
						}
					}
				}
			}

			if battery.FullmWh > 0 {
				result = append(result, battery)
			}
		}
	}

	return result, nil
}

func (b BatteryInfo) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Full", b.FullmWh, "mWh").
		AddField("Design", b.DesignmWh, "mWh").
		AddField("Current", b.CurrentmWh, "mWh").
		AddField("ChargePercent", b.ChargePercent, "%").
		AddField("State", b.State, "").
		Build()
}

func BatteryInfosToPrint(batteries []BatteryInfo) string {
	var sb strings.Builder
	for i, b := range batteries {
		sb.WriteString(fmt.Sprintf("  Battery[%d]:\n%s", i, b.ToPrint()))
		if i < len(batteries)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
