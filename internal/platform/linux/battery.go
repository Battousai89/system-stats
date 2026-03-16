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

// NewBatteryInfo gets battery information on Linux
func NewBatteryInfo() ([]types.BatteryInfo, error) {
	var result []types.BatteryInfo

	// Find all power supply devices
	matches, err := filepath.Glob("/sys/class/power_supply/BAT*")
	if err != nil {
		return nil, err
	}

	// Also check for BATT (some systems use this)
	battMatches, _ := filepath.Glob("/sys/class/power_supply/BATT*")
	matches = append(matches, battMatches...)

	for _, batPath := range matches {
		battery := parseBattery(batPath)
		if battery != nil {
			result = append(result, *battery)
		}
	}

	return result, nil
}

// parseBattery parses a battery device
func parseBattery(batPath string) *types.BatteryInfo {
	battery := &types.BatteryInfo{}

	// Read name
	if name, err := os.ReadFile(filepath.Join(batPath, "name")); err == nil {
		battery.Name = strings.TrimSpace(string(name))
	}

	// Read manufacturer
	if manufacturer, err := os.ReadFile(filepath.Join(batPath, "manufacturer")); err == nil {
		battery.Manufacturer = strings.TrimSpace(string(manufacturer))
	}

	// Read serial number
	if serial, err := os.ReadFile(filepath.Join(batPath, "serial_number")); err == nil {
		battery.SerialNumber = strings.TrimSpace(string(serial))
	}

	// Read model name
	if modelName, err := os.ReadFile(filepath.Join(batPath, "model_name")); err == nil {
		if battery.Name == "" {
			battery.Name = strings.TrimSpace(string(modelName))
		}
	}

	// Read chemistry
	if chemistry, err := os.ReadFile(filepath.Join(batPath, "chemistry")); err == nil {
		battery.Chemistry = strings.TrimSpace(string(chemistry))
	}

	// Read designed capacity
	if designedCap, err := os.ReadFile(filepath.Join(batPath, "energy_full_design")); err == nil {
		// In µWh
		cap, _ := strconv.ParseUint(strings.TrimSpace(string(designedCap)), 10, 64)
		battery.DesignedCapacity = cap / 1000 // Convert to mWh
	} else if designedCap, err := os.ReadFile(filepath.Join(batPath, "charge_full_design")); err == nil {
		// In µAh
		cap, _ := strconv.ParseUint(strings.TrimSpace(string(designedCap)), 10, 64)
		// Try to get voltage to convert to Wh
		if voltage, err := os.ReadFile(filepath.Join(batPath, "voltage_min_design")); err == nil {
			v, _ := strconv.ParseUint(strings.TrimSpace(string(voltage)), 10, 64)
			battery.DesignedCapacity = (cap * v) / 1000000000 // Convert to mWh
		} else {
			// Assume 12V nominal
			battery.DesignedCapacity = cap * 12 / 1000
		}
	}

	// Read full charge capacity
	if fullCap, err := os.ReadFile(filepath.Join(batPath, "energy_full")); err == nil {
		cap, _ := strconv.ParseUint(strings.TrimSpace(string(fullCap)), 10, 64)
		battery.FullChargeCap = cap / 1000 // Convert to mWh
	} else if fullCap, err := os.ReadFile(filepath.Join(batPath, "charge_full")); err == nil {
		cap, _ := strconv.ParseUint(strings.TrimSpace(string(fullCap)), 10, 64)
		if voltage, err := os.ReadFile(filepath.Join(batPath, "voltage_now")); err == nil {
			v, _ := strconv.ParseUint(strings.TrimSpace(string(voltage)), 10, 64)
			battery.FullChargeCap = (cap * v) / 1000000000
		} else {
			battery.FullChargeCap = cap * 12 / 1000
		}
	}

	// Read current capacity
	if currentCap, err := os.ReadFile(filepath.Join(batPath, "energy_now")); err == nil {
		cap, _ := strconv.ParseUint(strings.TrimSpace(string(currentCap)), 10, 64)
		battery.CurrentCapacity = cap / 1000 // Convert to mWh
	} else if currentCap, err := os.ReadFile(filepath.Join(batPath, "charge_now")); err == nil {
		cap, _ := strconv.ParseUint(strings.TrimSpace(string(currentCap)), 10, 64)
		if voltage, err := os.ReadFile(filepath.Join(batPath, "voltage_now")); err == nil {
			v, _ := strconv.ParseUint(strings.TrimSpace(string(voltage)), 10, 64)
			battery.CurrentCapacity = (cap * v) / 1000000000
		} else {
			battery.CurrentCapacity = cap * 12 / 1000
		}
	}

	// Read capacity percentage
	if percentStr, err := os.ReadFile(filepath.Join(batPath, "capacity")); err == nil {
		percent, _ := strconv.ParseUint(strings.TrimSpace(string(percentStr)), 10, 8)
		battery.Percent = uint8(percent)
	} else if battery.FullChargeCap > 0 {
		// Calculate percentage from capacity
		battery.Percent = uint8(float64(battery.CurrentCapacity) / float64(battery.FullChargeCap) * 100)
	}

	// Read status
	if status, err := os.ReadFile(filepath.Join(batPath, "status")); err == nil {
		battery.Status = strings.TrimSpace(string(status))
	}

	// Read voltage
	if voltage, err := os.ReadFile(filepath.Join(batPath, "voltage_now")); err == nil {
		v, _ := strconv.ParseUint(strings.TrimSpace(string(voltage)), 10, 64)
		battery.Voltage = uint32(v / 1000) // Convert from µV to mV
	}

	// Read time to empty/full
	if timeStr, err := os.ReadFile(filepath.Join(batPath, "time_to_empty_now")); err == nil {
		t, _ := strconv.ParseUint(strings.TrimSpace(string(timeStr)), 10, 32)
		if t > 0 && t < 1000000 { // Sanity check (in seconds)
			battery.EstTimeRemaining = uint32(t)
		}
	} else if timeStr, err := os.ReadFile(filepath.Join(batPath, "time_to_full_now")); err == nil {
		t, _ := strconv.ParseUint(strings.TrimSpace(string(timeStr)), 10, 32)
		if t > 0 && t < 1000000 {
			battery.EstTimeRemaining = uint32(t)
		}
	}

	// If we don't have time, try to calculate from power
	if battery.EstTimeRemaining == 0 && battery.CurrentCapacity > 0 {
		if power, err := os.ReadFile(filepath.Join(batPath, "power_now")); err == nil {
			p, _ := strconv.ParseUint(strings.TrimSpace(string(power)), 10, 64)
			if p > 0 && p < 100000000 { // Sanity check (in µW)
				// Time = Energy / Power
				energy := battery.CurrentCapacity * 1000000 // Convert mWh to µWh
				battery.EstTimeRemaining = uint32(energy / p)
			}
		} else if current, err := os.ReadFile(filepath.Join(batPath, "current_now")); err == nil {
			i, _ := strconv.ParseUint(strings.TrimSpace(string(current)), 10, 64)
			if i > 0 && i < 10000000 { // Sanity check (in µA)
				charge := battery.CurrentCapacity * 1000 // Convert mWh to µAh (assuming ~12V)
				battery.EstTimeRemaining = uint32(charge / i * 3600)
			}
		}
	}

	// Validate - skip if no meaningful data
	if battery.DesignedCapacity == 0 && battery.FullChargeCap == 0 && battery.CurrentCapacity == 0 {
		return nil
	}

	return battery
}

// GetAllBatteryStats collects all battery statistics
type AllBatteryStats struct {
	Batteries []types.BatteryInfo `json:"batteries"`
	Count     int                 `json:"count"`
}

// GetAllBatteryStats gets all battery information
func GetAllBatteryStats() (*AllBatteryStats, error) {
	batteries, err := NewBatteryInfo()
	if err != nil {
		return nil, err
	}

	return &AllBatteryStats{
		Batteries: batteries,
		Count:     len(batteries),
	}, nil
}
