//go:build linux
// +build linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"system-stats/internal/types"
)

// processStat holds parsed process stat information
type processStat struct {
	PID       int
	Comm      string
	State     byte
	PPID      int
	PGRP      int
	Session   int
	TTYNr     int
	TPGID     int
	Flags     uint
	MinFlt    uint
	CMinFlt   uint
	MajFlt    uint
	CMajFlt   uint
	UTime     uint
	STime     uint
	CUTime    int
	CSTime    int
	Priority  int
	Nice      int
	NumThreads int
	ItRealValue int
	StartTime uint
	VSize     uint
	RSS       int
}

// NewProcessInfo gets process information on Linux
func NewProcessInfo(topN int) ([]types.ProcessInfo, error) {
	// Get all process directories in /proc
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc: %w", err)
	}

	var processes []types.ProcessInfo
	totalMemory := getTotalMemory()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // Not a process directory
		}

		procInfo, err := readProcessInfo(pid, totalMemory)
		if err != nil {
			continue // Process may have exited
		}

		processes = append(processes, *procInfo)
	}

	// Sort by CPU usage
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPU > processes[j].CPU
	})

	// Take top N
	if topN > 0 && len(processes) > topN {
		processes = processes[:topN]
	}

	return processes, nil
}

// readProcessInfo reads information for a specific process
func readProcessInfo(pid int, totalMemory uint64) (*types.ProcessInfo, error) {
	// Read stat file
	stat, err := parseProcStat(pid)
	if err != nil {
		return nil, err
	}

	// Read status file for more details
	status, err := parseProcStatus(pid)
	if err != nil {
		status = make(map[string]string)
	}

	// Read command line
	cmdline, _ := readProcCmdline(pid)

	// Get process name
	name := stat.Comm
	if name == "" {
		name = status["Name"]
	}

	// Calculate CPU usage
	cpuPercent := calculateProcessCPU(pid, stat)

	// Get memory usage
	memory := uint64(stat.VSize)
	if memory == 0 {
		if rssStr, ok := status["VmRSS"]; ok {
			memory = parseMemoryValue(rssStr) * 1024 // Convert KB to bytes
		}
	}

	// Calculate memory percentage
	var memoryPercent float64
	if totalMemory > 0 && memory > 0 {
		memoryPercent = float64(memory) / float64(totalMemory) * 100.0
	}

	// Get process status
	processStatus := getProcessStatus(stat.State)

	// Get username
	username := getProcessUser(pid)

	// Get create time
	createTime := getProcessCreateTime(stat.StartTime)

	procInfo := &types.ProcessInfo{
		PID:         uint32(pid),
		Name:        name,
		CPU:         cpuPercent,
		Memory:      memory,
		MemoryPercent: memoryPercent,
		Status:      processStatus,
		Username:    username,
		Cmdline:     cmdline,
		CreateTime:  createTime,
		NumThreads:  uint32(stat.NumThreads),
	}

	return procInfo, nil
}

// parseProcStat parses /proc/[pid]/stat
func parseProcStat(pid int) (*processStat, error) {
	path := fmt.Sprintf("/proc/%d/stat", pid)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// The comm field can contain spaces and parentheses, so we need to handle it carefully
	line := string(content)
	
	// Find the last ')' which ends the comm field
	lastParen := strings.LastIndex(line, ")")
	if lastParen == -1 {
		return nil, fmt.Errorf("invalid stat format")
	}

	// Extract comm
	comm := line[1:lastParen] // Skip first '(' and exclude last ')'
	
	// Parse the rest of the fields
	rest := strings.Fields(line[lastParen+1:])
	if len(rest) < 20 {
		return nil, fmt.Errorf("not enough fields in stat")
	}

	stat := &processStat{
		PID:  pid,
		Comm: comm,
	}

	// Parse remaining fields
	if v, err := strconv.ParseUint(rest[0], 10, 8); err == nil {
		stat.State = byte(v)
	}
	stat.PPID, _ = strconv.Atoi(rest[1])
	stat.PGRP, _ = strconv.Atoi(rest[2])
	stat.Session, _ = strconv.Atoi(rest[3])
	stat.TTYNr, _ = strconv.Atoi(rest[4])
	stat.TPGID, _ = strconv.Atoi(rest[5])
	if v, err := strconv.ParseUint(rest[6], 10, 32); err == nil {
		stat.Flags = uint(v)
	}
	if v, err := strconv.ParseUint(rest[7], 10, 32); err == nil {
		stat.MinFlt = uint(v)
	}
	if v, err := strconv.ParseUint(rest[8], 10, 32); err == nil {
		stat.CMinFlt = uint(v)
	}
	if v, err := strconv.ParseUint(rest[9], 10, 32); err == nil {
		stat.MajFlt = uint(v)
	}
	if v, err := strconv.ParseUint(rest[10], 10, 32); err == nil {
		stat.CMajFlt = uint(v)
	}
	if v, err := strconv.ParseUint(rest[11], 10, 32); err == nil {
		stat.UTime = uint(v)
	}
	if v, err := strconv.ParseUint(rest[12], 10, 32); err == nil {
		stat.STime = uint(v)
	}
	stat.CUTime, _ = strconv.Atoi(rest[13])
	stat.CSTime, _ = strconv.Atoi(rest[14])
	stat.Priority, _ = strconv.Atoi(rest[15])
	stat.Nice, _ = strconv.Atoi(rest[16])
	stat.NumThreads, _ = strconv.Atoi(rest[17])
	stat.ItRealValue, _ = strconv.Atoi(rest[18])
	if v, err := strconv.ParseUint(rest[19], 10, 32); err == nil {
		stat.StartTime = uint(v)
	}
	if v, err := strconv.ParseUint(rest[20], 10, 32); err == nil {
		stat.VSize = uint(v)
	}
	stat.RSS, _ = strconv.Atoi(rest[21])

	return stat, nil
}

// parseProcStatus parses /proc/[pid]/status
func parseProcStatus(pid int) (map[string]string, error) {
	path := fmt.Sprintf("/proc/%d/status", pid)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		result[key] = value
	}

	return result, nil
}

// readProcCmdline reads /proc/[pid]/cmdline
func readProcCmdline(pid int) (string, error) {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// cmdline is null-separated
	cmdline := strings.ReplaceAll(string(content), "\x00", " ")
	return strings.TrimSpace(cmdline), nil
}

// calculateProcessCPU calculates CPU usage for a process
func calculateProcessCPU(pid int, stat *processStat) float64 {
	// Get system uptime
	uptime, err := getUptime()
	if err != nil {
		return 0
	}

	// Get clock ticks per second
	hz := getClockTicks()

	// Calculate process start time in seconds
	startTimeSec := float64(stat.StartTime) / float64(hz)

	// Calculate total CPU time used by process
	totalTime := float64(stat.UTime + stat.STime)

	// Calculate elapsed time since process start
	elapsedTime := float64(uptime) - startTimeSec

	if elapsedTime <= 0 {
		return 0
	}

	// Calculate CPU percentage
	cpuPercent := (totalTime / float64(hz)) / elapsedTime * 100.0

	return cpuPercent
}

// getProcessStatus converts process state to string
func getProcessStatus(state byte) string {
	switch state {
	case 'R':
		return "running"
	case 'S':
		return "sleeping"
	case 'D':
		return "disk sleep"
	case 'Z':
		return "zombie"
	case 'T':
		return "stopped"
	case 't':
		return "tracing stop"
	case 'W':
		return "paging"
	case 'X':
		return "dead"
	case 'x':
		return "dead"
	case 'K':
		return "wakekill"
	case 'P':
		return "parked"
	default:
		return "unknown"
	}
}

// getProcessUser gets the username for a process
func getProcessUser(pid int) string {
	// Read status file for Uid
	status, err := parseProcStatus(pid)
	if err != nil {
		return ""
	}

	uidStr, ok := status["Uid"]
	if !ok {
		return ""
	}

	// Get real UID (first number)
	parts := strings.Fields(uidStr)
	if len(parts) < 1 {
		return ""
	}

	uid, _ := strconv.Atoi(parts[0])

	// Try to get username from /etc/passwd
	return getUserNameByUID(uid)
}

// getUserNameByUID gets username from UID
func getUserNameByUID(uid int) string {
	content, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			if parts[2] == strconv.Itoa(uid) {
				return parts[0]
			}
		}
	}

	return ""
}

// getProcessCreateTime gets process creation time as Unix timestamp
func getProcessCreateTime(startTime uint) uint64 {
	hz := getClockTicks()
	startTimeSec := startTime / uint(hz)
	
	// Get current time
	now := uint64(time.Now().Unix())
	
	// Get system uptime
	uptime, _ := getUptime()
	
	// Calculate boot time
	bootTime := now - uptime
	
	return bootTime + uint64(startTimeSec)
}

// parseMemoryValue parses memory value string (e.g., "1234 kB") to KB
func parseMemoryValue(s string) uint64 {
	s = strings.TrimSpace(s)
	// Remove "kB" suffix if present
	s = strings.TrimSuffix(s, "kB")
	s = strings.TrimSpace(s)
	
	value, _ := strconv.ParseUint(s, 10, 64)
	return value
}

// getClockTicks gets the number of clock ticks per second
func getClockTicks() int {
	return 100 // Default value for most Linux systems (sysconf(_SC_CLK_TCK))
}

// getTotalMemory gets total system memory
func getTotalMemory() uint64 {
	content, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}

	memInfo := parseMemInfo(content)
	return memInfo["MemTotal"] * 1024 // Convert KB to bytes
}

// getProcessInfoByPID gets information for a specific process
func GetProcessInfoByPID(pid uint32) (*types.ProcessInfo, error) {
	totalMemory := getTotalMemory()
	return readProcessInfo(int(pid), totalMemory)
}

// GetAllProcessStats collects all process statistics
type AllProcessStats struct {
	Processes []types.ProcessInfo `json:"processes"`
	Total     int                 `json:"total"`
}

// GetAllProcessStats gets all process information
func GetAllProcessStats(topN int) (*AllProcessStats, error) {
	processes, err := NewProcessInfo(topN)
	if err != nil {
		return nil, err
	}

	return &AllProcessStats{
		Processes: processes,
		Total:     len(processes),
	}, nil
}
