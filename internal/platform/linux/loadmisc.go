//go:build linux
// +build linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"system-stats/internal/types"
)

// NewLoadMisc gets miscellaneous load information on Linux
func NewLoadMisc() (*types.LoadMisc, error) {
	info := &types.LoadMisc{}

	// Get uptime
	uptime, err := getUptime()
	if err == nil {
		info.Uptime = uptime
		info.UptimeDays = float64(uptime) / 86400.0
	}

	// Get boot time
	info.BootTime = getBootTime()

	// Get load average
	loadAvg, err := getLoadAverage()
	if err == nil {
		info.Load1 = loadAvg.Load1
		info.Load5 = loadAvg.Load5
		info.Load15 = loadAvg.Load15
	}

	// Get process and context switch info from /proc/stat
	if err := parseProcStatForMisc(info); err != nil {
		// Not critical, continue with defaults
	}

	// Get process counts
	info.ProcsTotal = uint64(getProcessCount())

	return info, nil
}

// getUptime gets system uptime in seconds
func getUptime() (uint64, error) {
	content, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(content))
	if len(fields) < 1 {
		return 0, fmt.Errorf("invalid uptime format")
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	return uint64(uptime), nil
}

// getBootTime gets system boot time as Unix timestamp
func getBootTime() uint64 {
	// Read btime from /proc/stat
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "btime") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				btime, _ := strconv.ParseUint(fields[1], 10, 64)
				return btime
			}
		}
	}

	// Fallback: calculate from uptime
	uptime, err := getUptime()
	if err != nil {
		return 0
	}

	return uint64(time.Now().Unix()) - uptime
}

// getLoadAverage gets load average from /proc/loadavg
func getLoadAverage() (*types.LoadAvg, error) {
	content, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(content))
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid loadavg format")
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return &types.LoadAvg{
		Load1:  load1,
		Load5:  load5,
		Load15: load15,
	}, nil
}

// parseProcStatForMisc parses /proc/stat for miscellaneous info
func parseProcStatForMisc(info *types.LoadMisc) error {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "ctxt":
			// Context switches
			info.ContextSwitches, _ = strconv.ParseUint(fields[1], 10, 64)
		case "processes":
			// Forks since boot
			// This is cumulative, not current running processes
		case "intr":
			// Interrupts
			info.Interrupts, _ = strconv.ParseUint(fields[1], 10, 64)
		}
	}

	return nil
}

// getProcessCount gets the number of processes
func getProcessCount() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name is a number (PID)
		if _, err := strconv.Atoi(entry.Name()); err == nil {
			count++
		}
	}

	return count
}

// GetProcessCount gets process count (exported function)
func GetProcessCount() (uint32, error) {
	return uint32(getProcessCount()), nil
}

// GetThreadCount gets thread count on Linux
func GetThreadCount() (uint32, error) {
	// Read from /proc/stat
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "procs_running") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				count, _ := strconv.ParseUint(fields[1], 10, 32)
				return uint32(count), nil
			}
		}
	}

	// Alternative: count threads in /proc
	return countThreads(), nil
}

// countThreads counts total threads
func countThreads() uint32 {
	count := uint32(0)
	
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		// Read status file for thread count
		statusPath := fmt.Sprintf("/proc/%d/status", pid)
		content, err := os.ReadFile(statusPath)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Threads:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					threads, _ := strconv.ParseUint(fields[1], 10, 32)
					count += uint32(threads)
				}
				break
			}
		}
	}

	return count
}

// GetUptime gets system uptime (exported function)
func GetUptime() (uint64, error) {
	return getUptime()
}

// GetLoadMiscStats collects all load miscellaneous statistics
type LoadMiscStats struct {
	LoadMisc *types.LoadMisc `json:"load_misc"`
}

// GetLoadMiscStats gets all load miscellaneous information
func GetLoadMiscStats() (*LoadMiscStats, error) {
	loadMisc, err := NewLoadMisc()
	if err != nil {
		return nil, err
	}

	return &LoadMiscStats{
		LoadMisc: loadMisc,
	}, nil
}
