package stats

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"system-stats/internal/formatter"
)

type ProcessInfo struct {
	PID        int32   `json:"pid"`
	PPID       int32   `json:"ppid"`
	Name       string  `json:"name"`
	CPU        float64 `json:"cpu"`
	Memory     float32 `json:"memory"`
	Status     string  `json:"status"`
	Username   string  `json:"username"`
	NumThreads int32   `json:"numThreads"`
}

func NewProcessInfo(topN int) ([]ProcessInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return getLinuxProcessInfo(topN)
	case "windows":
		return getWindowsProcessInfo(topN)
	case "darwin", "freebsd":
		return getUnixProcessInfo(topN)
	default:
		return []ProcessInfo{}, nil
	}
}

func getLinuxProcessInfo(topN int) ([]ProcessInfo, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var result []ProcessInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		info, err := parseProcessInfo(pid)
		if err != nil {
			continue
		}

		result = append(result, info)
	}

	sortProcessesByCPU(result)

	if topN > 0 && len(result) > topN {
		result = result[:topN]
	}

	return result, nil
}

func parseProcessInfo(pid int) (ProcessInfo, error) {
	info := ProcessInfo{PID: int32(pid)}

	statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return info, err
	}

	statStr := string(statData)
	startIdx := strings.Index(statStr, "(")
	endIdx := strings.LastIndex(statStr, ")")

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return info, nil
	}

	info.Name = statStr[startIdx+1 : endIdx]

	fields := strings.Fields(statStr[endIdx+1:])
	if len(fields) < 19 {
		return info, nil
	}

	ppid, _ := strconv.ParseInt(fields[1], 10, 32)
	info.PPID = int32(ppid)

	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)

	info.CPU = float64(utime+stime) / 100.0

	threads, _ := strconv.ParseInt(fields[17], 10, 32)
	info.NumThreads = int32(threads)

	info.Status = fields[0]

	statmData, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err == nil {
		statmFields := strings.Fields(string(statmData))
		if len(statmFields) >= 2 {
			rss, _ := strconv.ParseUint(statmFields[1], 10, 64)
			memBytes := rss * 4096
			totalMem, _ := getTotalSystemMemory()
			if totalMem > 0 {
				info.Memory = float32(float64(memBytes) / float64(totalMem) * 100)
			}
		}
	}

	info.Username = getProcessUser(pid)

	return info, nil
}

func getTotalSystemMemory() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.ParseUint(fields[1], 10, 64)
				return kb * 1024, nil
			}
		}
	}

	return 0, nil
}

func getProcessUser(pid int) string {
	statusData, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(string(statusData)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				uid := fields[1]
				if passwdData, err := os.ReadFile("/etc/passwd"); err == nil {
					for _, pline := range strings.Split(string(passwdData), "\n") {
						pfields := strings.Split(pline, ":")
						if len(pfields) >= 3 && pfields[2] == uid {
							return pfields[0]
						}
					}
				}
				return uid
			}
		}
	}

	return ""
}

func getWindowsProcessInfo(topN int) ([]ProcessInfo, error) {
	output, err := runCommandWithTimeout("wmic", "process", "get", "ProcessId,ParentProcessId,Name,WorkingSetSize,ThreadCount", "/format:csv")
	if err != nil {
		return []ProcessInfo{}, nil
	}

	var result []ProcessInfo
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			continue
		}

		pid, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 32)
		ppid, _ := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 32)
		mem, _ := strconv.ParseUint(strings.TrimSpace(parts[4]), 10, 64)
		threads, _ := strconv.ParseInt(strings.TrimSpace(parts[5]), 10, 32)

		totalMem, _ := getTotalWindowsMemory()
		memPercent := float32(0)
		if totalMem > 0 {
			memPercent = float32(float64(mem) / float64(totalMem) * 100)
		}

		result = append(result, ProcessInfo{
			PID:        int32(pid),
			PPID:       int32(ppid),
			Name:       strings.TrimSpace(parts[3]),
			Memory:     memPercent,
			NumThreads: int32(threads),
		})
	}

	sortProcessesByCPU(result)

	if topN > 0 && len(result) > topN {
		result = result[:topN]
	}

	return result, nil
}

func getTotalWindowsMemory() (uint64, error) {
	output, err := runCommandWithTimeout("wmic", "OS", "get", "TotalVisibleMemorySize", "/format:csv")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		kb, _ := strconv.ParseUint(strings.TrimSpace(line), 10, 64)
		return kb * 1024, nil
	}

	return 0, nil
}

func getUnixProcessInfo(topN int) ([]ProcessInfo, error) {
	output, err := runCommandWithTimeout("ps", "aux")
	if err != nil {
		return []ProcessInfo{}, nil
	}

	var result []ProcessInfo
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		pid, _ := strconv.ParseInt(fields[1], 10, 32)
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		threads, _ := strconv.ParseInt(fields[10], 10, 32)

		result = append(result, ProcessInfo{
			PID:        int32(pid),
			Name:       fields[10],
			CPU:        cpu,
			Memory:     float32(mem),
			NumThreads: int32(threads),
			Username:   fields[0],
		})
	}

	if topN > 0 && len(result) > topN {
		result = result[:topN]
	}

	return result, nil
}

func sortProcessesByCPU(procs []ProcessInfo) {
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].CPU > procs[j].CPU
	})
}

func (p ProcessInfo) ToPrint() string {
	return formatter.NewBuilder().
		AddField("PID", p.PID, "").
		AddField("PPID", p.PPID, "").
		AddField("Name", p.Name, "").
		AddField("CPU", p.CPU, "%").
		AddField("Memory", p.Memory, "%").
		AddField("Status", p.Status, "").
		AddField("User", p.Username, "").
		AddField("Threads", p.NumThreads, "").
		Build()
}

func ProcessInfosToPrint(procs []ProcessInfo) string {
	var sb strings.Builder
	for i, p := range procs {
		sb.WriteString(fmt.Sprintf("  [%d] ", i+1))
		sb.WriteString(p.ToPrint())
		if i < len(procs)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
