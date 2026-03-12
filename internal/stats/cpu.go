package stats

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"system-stats/internal/config"
	"system-stats/internal/formatter"
)

type CPUInfo struct {
	CPU         int32    `json:"cpu"`
	VendorID    string   `json:"vendorID"`
	Family      string   `json:"family"`
	Model       string   `json:"model"`
	Stepping    int32    `json:"stepping"`
	PhysicalID  string   `json:"physicalID"`
	CoreID      string   `json:"coreID"`
	Cores       int32    `json:"cores"`
	ModelName   string   `json:"modelName"`
	CacheSizeKB int32    `json:"cacheSizeKB"`
	Flags       int      `json:"flags"`
	FlagsList   []string `json:"flagsList,omitempty"`
	Microcode   string   `json:"microcode"`
	Mhz         float64  `json:"mhz"`
}

type CPUTimes struct {
	CPU       string  `json:"cpu"`
	User      float64 `json:"user"`
	System    float64 `json:"system"`
	Idle      float64 `json:"idle"`
	Nice      float64 `json:"nice"`
	Iowait    float64 `json:"iowait"`
	Irq       float64 `json:"irq"`
	Softirq   float64 `json:"softirq"`
	Steal     float64 `json:"steal"`
	Guest     float64 `json:"guest"`
	GuestNice float64 `json:"guestNice"`
	Total     float64 `json:"total"`
}

type CPUPercent struct {
	CPU     string  `json:"cpu"`
	Percent float64 `json:"percent"`
}

func NewCPUInfo() ([]CPUInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return parseProcCpuinfo()
	case "windows":
		return getWindowsCPUInfo()
	case "darwin":
		return getDarwinCPUInfo()
	default:
		return getGenericCPUInfo()
	}
}

func parseProcCpuinfo() ([]CPUInfo, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []CPUInfo
	var current CPUInfo

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if current.ModelName != "" {
				result = append(result, current)
				current = CPUInfo{}
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
			cpu, _ := strconv.ParseInt(value, 10, 32)
			current.CPU = int32(cpu)
		case "vendor_id":
			current.VendorID = value
		case "cpu family":
			current.Family = value
		case "model":
			current.Model = value
		case "stepping":
			stepping, _ := strconv.ParseInt(value, 10, 32)
			current.Stepping = int32(stepping)
		case "physical id":
			current.PhysicalID = value
		case "core id":
			current.CoreID = value
		case "cpu cores":
			cores, _ := strconv.ParseInt(value, 10, 32)
			current.Cores = int32(cores)
		case "model name":
			current.ModelName = value
		case "cache size":
			if value != "" && value != "0 KB" {
				cache, _ := strconv.ParseInt(strings.TrimSuffix(value, " KB"), 10, 32)
				current.CacheSizeKB = int32(cache)
			}
		case "flags":
			flags := strings.Fields(value)
			current.Flags = len(flags)
			current.FlagsList = flags
		case "microcode":
			current.Microcode = value
		case "cpu MHz":
			mhz, _ := strconv.ParseFloat(value, 64)
			current.Mhz = mhz
		}
	}

	if current.ModelName != "" {
		result = append(result, current)
	}

	return result, scanner.Err()
}

func getWindowsCPUInfo() ([]CPUInfo, error) {
	output, err := runCommandWithTimeout("wmic", "cpu", "get", "Name,NumberOfCores,NumberOfLogicalProcessors,Manufacturer,MaxClockSpeed,DeviceID", "/format:csv")
	if err != nil {
		return nil, err
	}

	var result []CPUInfo
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}

		cores, _ := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 32)
		mhz, _ := strconv.ParseFloat(strings.TrimSpace(parts[5]), 64)

		result = append(result, CPUInfo{
			CPU:       int32(i - 1),
			ModelName: strings.TrimSpace(parts[1]),
			Cores:     int32(cores),
			VendorID:  strings.TrimSpace(parts[4]),
			Mhz:       mhz,
		})
	}

	return result, nil
}

func getDarwinCPUInfo() ([]CPUInfo, error) {
	output, err := runCommandWithTimeout("sysctl", "-a")
	if err != nil {
		return nil, err
	}

	var info CPUInfo
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "machdep.cpu.brand_string:") {
			info.ModelName = strings.TrimSpace(strings.TrimPrefix(line, "machdep.cpu.brand_string:"))
		} else if strings.HasPrefix(line, "machdep.cpu.core_count:") {
			cores, _ := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "machdep.cpu.core_count:")), 10, 32)
			info.Cores = int32(cores)
		} else if strings.HasPrefix(line, "hw.ncpu:") {
			cpu, _ := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "hw.ncpu:")), 10, 32)
			info.CPU = int32(cpu)
		}
	}

	return []CPUInfo{info}, nil
}

func getGenericCPUInfo() ([]CPUInfo, error) {
	return []CPUInfo{
		{
			CPU:       0,
			Cores:     int32(runtime.NumCPU()),
			ModelName: runtime.GOARCH,
		},
	}, nil
}

func NewCPUTimes() ([]CPUTimes, error) {
	switch runtime.GOOS {
	case "linux":
		return parseProcStat()
	case "windows":
		return getWindowsCPUTimes()
	case "darwin", "freebsd":
		return getUnixCPUTimes()
	default:
		return getGenericCPUTimes()
	}
}

func parseProcStat() ([]CPUTimes, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []CPUTimes
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		cpu := fields[0]
		user, _ := strconv.ParseFloat(fields[1], 64)
		nice, _ := strconv.ParseFloat(fields[2], 64)
		system, _ := strconv.ParseFloat(fields[3], 64)
		idle, _ := strconv.ParseFloat(fields[4], 64)
		iowait, _ := strconv.ParseFloat(fields[5], 64)
		irq, _ := strconv.ParseFloat(fields[6], 64)
		softirq, _ := strconv.ParseFloat(fields[7], 64)
		steal := 0.0
		guest := 0.0
		guestNice := 0.0

		if len(fields) > 8 {
			steal, _ = strconv.ParseFloat(fields[8], 64)
		}
		if len(fields) > 9 {
			guest, _ = strconv.ParseFloat(fields[9], 64)
		}
		if len(fields) > 10 {
			guestNice, _ = strconv.ParseFloat(fields[10], 64)
		}

		total := user + nice + system + idle + iowait + irq + softirq + steal + guest + guestNice

		result = append(result, CPUTimes{
			CPU:       cpu,
			User:      user,
			Nice:      nice,
			System:    system,
			Idle:      idle,
			Iowait:    iowait,
			Irq:       irq,
			Softirq:   softirq,
			Steal:     steal,
			Guest:     guest,
			GuestNice: guestNice,
			Total:     total,
		})
	}

	return result, scanner.Err()
}

func getWindowsCPUTimes() ([]CPUTimes, error) {
	output, err := runCommandWithTimeout("wmic", "cpu", "get", "LoadPercentage", "/format:csv")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var totalLoad float64
	var count int

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		load, _ := strconv.ParseFloat(strings.TrimSpace(line), 64)
		totalLoad += load
		count++
	}

	if count == 0 {
		count = 1
	}

	avgLoad := totalLoad / float64(count)

	return []CPUTimes{
		{
			CPU:    "cpu",
			User:   avgLoad,
			System: 0,
			Idle:   100 - avgLoad,
			Total:  100,
		},
	}, nil
}

func getUnixCPUTimes() ([]CPUTimes, error) {
	output, err := runCommandWithTimeout("top", "-l", "1")
	if err != nil {
		return getGenericCPUTimes()
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "CPU usage:") {
			parts := strings.Split(line, ",")
			var user, system, idle float64
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.Contains(p, "user") {
					fmt.Sscanf(p, "%f", &user)
				} else if strings.Contains(p, "sys") {
					fmt.Sscanf(p, "%f", &system)
				} else if strings.Contains(p, "idle") {
					fmt.Sscanf(p, "%f", &idle)
				}
			}
			return []CPUTimes{
				{
					CPU:    "cpu",
					User:   user,
					System: system,
					Idle:   idle,
					Total:  user + system + idle,
				},
			}, nil
		}
	}

	return getGenericCPUTimes()
}

func getGenericCPUTimes() ([]CPUTimes, error) {
	return []CPUTimes{
		{
			CPU:    "cpu",
			User:   0,
			System: 0,
			Idle:   100,
			Total:  100,
		},
	}, nil
}

func NewCPUPercent() ([]CPUPercent, error) {
	times1, err := NewCPUTimes()
	if err != nil {
		return nil, err
	}

	time.Sleep(config.CPUSamplingInterval)

	times2, err := NewCPUTimes()
	if err != nil {
		return nil, err
	}

	var result []CPUPercent
	for i, t1 := range times1 {
		if i >= len(times2) {
			break
		}
		t2 := times2[i]

		total1 := t1.Total
		total2 := t2.Total
		totalDiff := total2 - total1

		if totalDiff == 0 {
			totalDiff = 1
		}

		idle1 := t1.Idle
		idle2 := t2.Idle
		idleDiff := idle2 - idle1

		percent := ((totalDiff - idleDiff) / totalDiff) * 100
		if percent < 0 {
			percent = 0
		}

		result = append(result, CPUPercent{
			CPU:     t1.CPU,
			Percent: percent,
		})
	}

	return result, nil
}

func (info CPUInfo) ToPrint() string {
	return formatter.NewBuilder().
		AddField("CPU", info.CPU, "").
		AddField("VendorID", info.VendorID, "").
		AddField("Family", info.Family, "").
		AddField("Model", info.Model, "").
		AddField("Stepping", info.Stepping, "").
		AddField("PhysicalID", info.PhysicalID, "").
		AddField("CoreID", info.CoreID, "").
		AddField("Cores", info.Cores, "").
		AddField("ModelName", info.ModelName, "").
		AddField("Mhz", info.Mhz, "MHz").
		AddField("CacheSize", info.CacheSizeKB, "KB").
		AddField("Flags", info.Flags, "").
		AddField("Microcode", info.Microcode, "").
		Build()
}

func (t CPUTimes) ToPrint() string {
	return formatter.NewBuilder().
		AddField("CPU", t.CPU, "").
		AddField("User", t.User, "").
		AddField("System", t.System, "").
		AddField("Idle", t.Idle, "").
		AddField("Nice", t.Nice, "").
		AddField("Iowait", t.Iowait, "").
		AddField("Irq", t.Irq, "").
		AddField("Softirq", t.Softirq, "").
		AddField("Steal", t.Steal, "").
		AddField("Guest", t.Guest, "").
		AddField("GuestNice", t.GuestNice, "").
		AddField("Total", t.Total, "").
		Build()
}

func (p CPUPercent) ToPrint() string {
	return formatter.NewBuilder().
		AddField("CPU", p.CPU, "").
		AddField("Usage", p.Percent, "%").
		Build()
}

func CPUInfosToPrint(infos []CPUInfo) string {
	var sb strings.Builder
	for i, info := range infos {
		sb.WriteString(fmt.Sprintf("  CPU[%d]:\n%s", i, info.ToPrint()))
		if i < len(infos)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func CPUTimesToPrint(times []CPUTimes) string {
	var sb strings.Builder
	first := true
	for i, t := range times {
		if t.CPU == "cpu" && len(times) > 1 {
			continue
		}
		if !first {
			sb.WriteString("\n")
		}
		first = false
		sb.WriteString(fmt.Sprintf("  CPU[%d]:\n%s", i, t.ToPrint()))
	}
	return sb.String()
}

func CPUPercentsToPrint(percents []CPUPercent) string {
	var sb strings.Builder
	first := true
	for i, p := range percents {
		if p.CPU == "cpu" && len(percents) > 1 {
			continue
		}
		if !first {
			sb.WriteString("\n")
		}
		first = false
		sb.WriteString(fmt.Sprintf("  CPU[%d]:\n%s", i, p.ToPrint()))
	}
	return sb.String()
}
