//go:build linux
// +build linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"system-stats/internal/config"
	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// cpuInfo holds parsed CPU information
type cpuInfo struct {
	Processor         uint32
	VendorID          string
	CPUFamily         string
	Model             string
	ModelName         string
	Stepping          string
	Microcode         string
	CPUMHz            float64
	CacheSize         string
	PhysicalID        string
	Siblings          uint32
	CoreID            string
	CPUCores          uint32
	ApicID            string
	InitialApicID     string
	FPU               string
	FPUException      string
	CPUIDLevel        uint32
	WP                string
	Flags             []string
	Bugs              []string
	BogoMIPS          float64
	CLFlushSize       uint32
	CacheAlignment    uint32
	AddressSizes      string
	PowerManagement   string
}

// NewCPUInfo gets CPU information on Linux
func NewCPUInfo() ([]types.CPUInfo, error) {
	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/cpuinfo: %w", err)
	}

	cpus := parseCPUInfo(content)
	
	// Get temperature if available
	temp := getCPUTemperature()

	// Group by physical CPU
	physicalCPUs := make(map[string][]cpuInfo)
	for _, cpu := range cpus {
		physicalCPUs[cpu.PhysicalID] = append(physicalCPUs[cpu.PhysicalID], cpu)
	}

	result := make([]types.CPUInfo, 0, len(physicalCPUs))
	for _, cpuGroup := range physicalCPUs {
		if len(cpuGroup) == 0 {
			continue
		}

		first := cpuGroup[0]
		
		// Calculate total cores and threads
		uniqueCores := make(map[string]bool)
		totalThreads := uint32(len(cpuGroup))
		for _, cpu := range cpuGroup {
			uniqueCores[cpu.CoreID] = true
		}
		totalCores := uint32(len(uniqueCores))
		if totalCores == 0 {
			totalCores = totalThreads
		}

		// Get cache size from first CPU
		cacheSize := parseCacheSize(first.CacheSize)

		// Get current clock speed from sysfs if available
		clockSpeed := getCPUClockSpeed(0)
		if clockSpeed == 0 {
			clockSpeed = uint64(first.CPUMHz * 1000) // Convert MHz to kHz
		}

		cpuInfo := types.CPUInfo{
			Name:                first.ModelName,
			Manufacturer:        getCPUVendor(first.VendorID),
			Family:              first.CPUFamily,
			Model:               first.Model,
			Stepping:            first.Stepping,
			Architecture:        getCPUArchitecture(),
			Cores:               totalCores,
			LogicalProcessors:   totalThreads,
			CurrentClockSpeed:   clockSpeed / 1000, // Convert to MHz
			MaxClockSpeed:       getMaxCPUClockSpeed(),
			LoadPercentage:      uint8(getCPULoadPercentage()),
			Status:              "OK",
			Enabled:             true,
			NumberOfCores:       totalCores,
			NumberOfLogicalProcessors: totalThreads,
		}

		if cacheSize > 0 {
			cpuInfo.L2CacheSize = cacheSize
		}

		if temp > 0 {
			cpuInfo.Temperature = uint32(temp)
		}

		result = append(result, cpuInfo)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no CPU information found")
	}

	return result, nil
}

// NewCPUTimes gets CPU times on Linux
func NewCPUTimes() ([]types.CPUTimes, error) {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/stat: %w", err)
	}

	return parseCPUTimes(content)
}

// NewCPUPercent gets CPU usage percentage on Linux
func NewCPUPercent() ([]types.CPUPercent, error) {
	// Take two samples with sampling interval
	sample1, err := readCPUStats()
	if err != nil {
		return nil, err
	}

	time.Sleep(config.CPUSamplingInterval)

	sample2, err := readCPUStats()
	if err != nil {
		return nil, err
	}

	return calculateCPUPercent(sample1, sample2)
}

// parseCPUInfo parses /proc/cpuinfo content
func parseCPUInfo(content []byte) []cpuInfo {
	var cpus []cpuInfo
	var current cpuInfo

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		
		if line == "" {
			if current.ModelName != "" {
				cpus = append(cpus, current)
				current = cpuInfo{}
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "processor":
			if v, err := strconv.ParseUint(value, 10, 32); err == nil {
				current.Processor = uint32(v)
			}
		case "vendor_id":
			current.VendorID = value
		case "cpu family":
			current.CPUFamily = value
		case "model":
			current.Model = value
		case "model name":
			current.ModelName = value
		case "stepping":
			current.Stepping = value
		case "microcode":
			current.Microcode = value
		case "cpu MHz":
			current.CPUMHz, _ = strconv.ParseFloat(value, 64)
		case "cache size":
			current.CacheSize = value
		case "physical id":
			current.PhysicalID = value
		case "siblings":
			if v, err := strconv.ParseUint(value, 10, 32); err == nil {
				current.Siblings = uint32(v)
			}
		case "core id":
			current.CoreID = value
		case "cpu cores":
			if v, err := strconv.ParseUint(value, 10, 32); err == nil {
				current.CPUCores = uint32(v)
			}
		case "apicid":
			current.ApicID = value
		case "initial apicid":
			current.InitialApicID = value
		case "fpu":
			current.FPU = value
		case "fpu_exception":
			current.FPUException = value
		case "cpuid level":
			if v, err := strconv.ParseUint(value, 10, 32); err == nil {
				current.CPUIDLevel = uint32(v)
			}
		case "wp":
			current.WP = value
		case "flags":
			current.Flags = strings.Fields(value)
		case "bugs":
			current.Bugs = strings.Fields(value)
		case "bogomips":
			current.BogoMIPS, _ = strconv.ParseFloat(value, 64)
		case "clflush size":
			if v, err := strconv.ParseUint(value, 10, 32); err == nil {
				current.CLFlushSize = uint32(v)
			}
		case "cache_alignment":
			if v, err := strconv.ParseUint(value, 10, 32); err == nil {
				current.CacheAlignment = uint32(v)
			}
		case "address sizes":
			current.AddressSizes = value
		case "power management":
			current.PowerManagement = value
		}
	}

	if current.ModelName != "" {
		cpus = append(cpus, current)
	}

	return cpus
}

// parseCPUTimes parses /proc/stat and returns CPU times
func parseCPUTimes(content []byte) ([]types.CPUTimes, error) {
	var result []types.CPUTimes

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		cpuName := fields[0]
		
		// Parse CPU time values (in USER_HZ units, typically 100Hz)
		user, _ := strconv.ParseFloat(fields[1], 64)
		nice, _ := strconv.ParseFloat(fields[2], 64)
		system, _ := strconv.ParseFloat(fields[3], 64)
		idle, _ := strconv.ParseFloat(fields[4], 64)
		iowait, _ := strconv.ParseFloat(fields[5], 64)
		irq, _ := strconv.ParseFloat(fields[6], 64)
		softirq, _ := strconv.ParseFloat(fields[7], 64)
		
		// Convert from centiseconds to seconds
		userSec := user / 100.0
		systemSec := system / 100.0
		idleSec := idle / 100.0
		iowaitSec := iowait / 100.0
		irqSec := irq / 100.0
		softirqSec := softirq / 100.0

		total := userSec + nice/100.0 + systemSec + idleSec + iowaitSec + irqSec + softirqSec
		
		usage := 0.0
		if total > 0 {
			usage = (userSec + systemSec) / total * 100.0
		}

		cpuTime := types.CPUTimes{
			CPU:       cpuName,
			User:      userSec,
			System:    systemSec,
			Idle:      idleSec,
			Interrupt: irqSec + softirqSec,
			DPC:       0, // Linux doesn't have DPC
			Total:     total,
			Usage:     usage,
		}

		result = append(result, cpuTime)
	}

	return result, nil
}

// cpuStats holds CPU statistics for percentage calculation
type cpuStats struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

// readCPUStats reads current CPU stats
func readCPUStats() (map[string]cpuStats, error) {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil, err
	}

	result := make(map[string]cpuStats)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		stats := cpuStats{}
		stats.user, _ = strconv.ParseUint(fields[1], 10, 64)
		stats.nice, _ = strconv.ParseUint(fields[2], 10, 64)
		stats.system, _ = strconv.ParseUint(fields[3], 10, 64)
		stats.idle, _ = strconv.ParseUint(fields[4], 10, 64)
		stats.iowait, _ = strconv.ParseUint(fields[5], 10, 64)
		stats.irq, _ = strconv.ParseUint(fields[6], 10, 64)
		stats.softirq, _ = strconv.ParseUint(fields[7], 10, 64)
		if len(fields) > 8 {
			stats.steal, _ = strconv.ParseUint(fields[8], 10, 64)
		}

		result[fields[0]] = stats
	}

	return result, nil
}

// calculateCPUPercent calculates CPU percentage from two samples
func calculateCPUPercent(sample1, sample2 map[string]cpuStats) ([]types.CPUPercent, error) {
	var result []types.CPUPercent

	for cpuName, stats2 := range sample2 {
		stats1, exists := sample1[cpuName]
		if !exists {
			continue
		}

		// Calculate deltas
		userDelta := float64(stats2.user - stats1.user)
		niceDelta := float64(stats2.nice - stats1.nice)
		systemDelta := float64(stats2.system - stats1.system)
		idleDelta := float64(stats2.idle - stats1.idle)
		iowaitDelta := float64(stats2.iowait - stats1.iowait)
		irqDelta := float64(stats2.irq - stats1.irq)
		softirqDelta := float64(stats2.softirq - stats1.softirq)
		stealDelta := float64(stats2.steal - stats1.steal)

		totalDelta := userDelta + niceDelta + systemDelta + idleDelta + iowaitDelta + irqDelta + softirqDelta + stealDelta

		var percent, userPercent, systemPercent, idlePercent float64
		if totalDelta > 0 {
			percent = (userDelta + niceDelta + systemDelta + irqDelta + softirqDelta + stealDelta) / totalDelta * 100.0
			userPercent = (userDelta + niceDelta) / totalDelta * 100.0
			systemPercent = (systemDelta + irqDelta + softirqDelta) / totalDelta * 100.0
			idlePercent = (idleDelta + iowaitDelta) / totalDelta * 100.0
		}

		if cpuName != "cpu" { // Skip total CPU
			result = append(result, types.CPUPercent{
				CPU:           cpuName,
				Percent:       percent,
				UserPercent:   userPercent,
				SystemPercent: systemPercent,
				IdlePercent:   idlePercent,
			})
		}
	}

	// Sort by CPU name
	sort.Slice(result, func(i, j int) bool {
		return result[i].CPU < result[j].CPU
	})

	return result, nil
}

// getCPUVendor returns vendor name from vendor ID
func getCPUVendor(vendorID string) string {
	switch vendorID {
	case "GenuineIntel":
		return "Intel"
	case "AuthenticAMD":
		return "AMD"
	case "ARM":
		return "ARM"
	default:
		return vendorID
	}
}

// getCPUTemperature reads CPU temperature from hwmon
func getCPUTemperature() float64 {
	// Try thermal zones first
	if temp, err := readThermalZone(); err == nil && temp > 0 {
		return temp
	}

	// Try hwmon
	if temp, err := readHwmonTemp(); err == nil && temp > 0 {
		return temp
	}

	return 0
}

// readThermalZone reads temperature from thermal zone
func readThermalZone() (float64, error) {
	files, err := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	if err != nil {
		return 0, err
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		temp, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
		if err != nil {
			continue
		}

		// Temperature is in millidegrees Celsius
		temp = temp / 1000.0
		
		// Sanity check (reasonable CPU temperature)
		if temp > 0 && temp < 150 {
			return temp, nil
		}
	}

	return 0, fmt.Errorf("no valid thermal zone found")
}

// readHwmonTemp reads temperature from hwmon
func readHwmonTemp() (float64, error) {
	files, err := filepath.Glob("/sys/class/hwmon/hwmon*/temp1_input")
	if err != nil {
		return 0, err
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		temp, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
		if err != nil {
			continue
		}

		// Temperature is in millidegrees Celsius
		temp = temp / 1000.0
		
		if temp > 0 && temp < 150 {
			return temp, nil
		}
	}

	return 0, fmt.Errorf("no valid hwmon sensor found")
}

// getCPUClockSpeed gets current CPU clock speed from sysfs
func getCPUClockSpeed(cpuNum int) uint64 {
	// Try to read from cpufreq
	path := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/scaling_cur_freq", cpuNum)
	content, err := os.ReadFile(path)
	if err != nil {
		// Try alternative path
		path = fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/cpuinfo_cur_freq", cpuNum)
		content, err = os.ReadFile(path)
		if err != nil {
			return 0
		}
	}

	freq, err := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
	if err != nil {
		return 0
	}

	// Frequency is in kHz
	return freq
}

// getMaxCPUClockSpeed gets maximum CPU clock speed
func getMaxCPUClockSpeed() uint64 {
	path := "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq"
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	freq, err := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
	if err != nil {
		return 0
	}

	return freq / 1000 // Convert to MHz
}

// getCPULoadPercentage gets current CPU load percentage
func getCPULoadPercentage() float64 {
	sample1, err := readCPUStats()
	if err != nil {
		return 0
	}

	time.Sleep(100 * time.Millisecond)

	sample2, err := readCPUStats()
	if err != nil {
		return 0
	}

	percents, err := calculateCPUPercent(sample1, sample2)
	if err != nil || len(percents) == 0 {
		return 0
	}

	// Return average across all cores
	var total float64
	for _, p := range percents {
		total += p.Percent
	}
	return total / float64(len(percents))
}

// getCPUArchitecture returns system architecture
func getCPUArchitecture() string {
	arch, _ := getArchitectureFromUname()
	return arch
}

// getArchitectureFromUname gets architecture from uname command
func getArchitectureFromUname() (string, error) {
	output, err := helpers.RunCommandWithTimeout("uname", "-m")
	if err != nil {
		return "unknown", err
	}
	return strings.TrimSpace(string(output)), nil
}

// parseCacheSize parses cache size string to KB
func parseCacheSize(cacheSize string) uint32 {
	if cacheSize == "" {
		return 0
	}

	re := regexp.MustCompile(`(\d+)\s*(KB|MB|GB)?`)
	matches := re.FindStringSubmatch(cacheSize)
	if len(matches) < 2 {
		return 0
	}

	value, err := strconv.ParseUint(matches[1], 10, 32)
	if err != nil {
		return 0
	}

	unit := "KB"
	if len(matches) > 2 && matches[2] != "" {
		unit = matches[2]
	}

	switch unit {
	case "MB":
		return uint32(value) * 1024
	case "GB":
		return uint32(value) * 1024 * 1024
	default:
		return uint32(value)
	}
}

// GetCPUCoreCount returns the number of CPU cores
func GetCPUCoreCount() (uint32, error) {
	info, err := NewCPUInfo()
	if err != nil {
		return 0, err
	}

	var totalCores uint32
	for _, cpu := range info {
		totalCores += cpu.Cores
	}
	return totalCores, nil
}

// GetCPUThreadCount returns the number of CPU threads
func GetCPUThreadCount() (uint32, error) {
	info, err := NewCPUInfo()
	if err != nil {
		return 0, err
	}

	var totalThreads uint32
	for _, cpu := range info {
		totalThreads += cpu.LogicalProcessors
	}
	return totalThreads, nil
}

// GetCPUModelName returns the CPU model name
func GetCPUModelName() (string, error) {
	info, err := NewCPUInfo()
	if err != nil {
		return "", err
	}

	if len(info) > 0 {
		return info[0].Name, nil
	}
	return "", nil
}

// AllCPUStats holds all CPU statistics
type AllCPUStats struct {
	Info        []types.CPUInfo   `json:"info"`
	Times       []types.CPUTimes  `json:"times"`
	Percent     []types.CPUPercent `json:"percent"`
	CoreCount   uint32            `json:"core_count"`
	ThreadCount uint32            `json:"thread_count"`
}

// GetAllCPUStats collects all CPU statistics
func GetAllCPUStats() (*AllCPUStats, error) {
	result := &AllCPUStats{}

	info, err := NewCPUInfo()
	if err != nil {
		return nil, err
	}
	result.Info = info

	times, err := NewCPUTimes()
	if err != nil {
		return nil, err
	}
	result.Times = times

	percent, err := NewCPUPercent()
	if err != nil {
		return nil, err
	}
	result.Percent = percent

	var totalCores, totalThreads uint32
	for _, cpu := range info {
		totalCores += cpu.Cores
		totalThreads += cpu.LogicalProcessors
	}
	result.CoreCount = totalCores
	result.ThreadCount = totalThreads

	return result, nil
}
