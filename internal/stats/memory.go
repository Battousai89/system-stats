package stats

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"

	"system-stats/internal/formatter"
)

type VirtualMemory struct {
	TotalBytes     uint64  `json:"totalBytes"`
	AvailableBytes uint64  `json:"availableBytes"`
	UsedBytes      uint64  `json:"usedBytes"`
	UsedPercent    float32 `json:"usedPercent"`
	FreeBytes      uint64  `json:"freeBytes"`
	ActiveBytes    uint64  `json:"activeBytes"`
	InactiveBytes  uint64  `json:"inactiveBytes"`
	WiredBytes     uint64  `json:"wiredBytes"`
	LaundryBytes   uint64  `json:"laundryBytes"`
	BuffersBytes   uint64  `json:"buffersBytes"`
	CachedBytes    uint64  `json:"cachedBytes"`
	SharedBytes    uint64  `json:"sharedBytes"`
	SlabBytes      uint64  `json:"slabBytes"`
	SwapTotalBytes uint64  `json:"swapTotalBytes"`
	SwapFreeBytes  uint64  `json:"swapFreeBytes"`
	SwapUsedBytes  uint64  `json:"swapUsedBytes"`
}

type VirtualMemoryStat struct {
	Total       uint64
	Available   uint64
	Used        uint64
	UsedPercent float64
	Free        uint64
	Active      uint64
	Inactive    uint64
	Wired       uint64
	Laundry     uint64
	Buffers     uint64
	Cached      uint64
	Shared      uint64
	Slab        uint64
	Sreclaimable uint64
	Sunreclaim  uint64
	SwapTotal   uint64
	SwapFree    uint64
}

func NewVirtualMemory(vm *VirtualMemoryStat) *VirtualMemory {
	swapUsed := vm.SwapTotal - vm.SwapFree
	return &VirtualMemory{
		TotalBytes:     vm.Total,
		AvailableBytes: vm.Available,
		UsedBytes:      vm.Used,
		UsedPercent:    float32(vm.UsedPercent),
		FreeBytes:      vm.Free,
		ActiveBytes:    vm.Active,
		InactiveBytes:  vm.Inactive,
		WiredBytes:     vm.Wired,
		LaundryBytes:   vm.Laundry,
		BuffersBytes:   vm.Buffers,
		CachedBytes:    vm.Cached,
		SharedBytes:    vm.Shared,
		SlabBytes:      vm.Slab,
		SwapTotalBytes: vm.SwapTotal,
		SwapFreeBytes:  vm.SwapFree,
		SwapUsedBytes:  swapUsed,
	}
}

func GetVirtualMemory() (*VirtualMemoryStat, error) {
	switch runtime.GOOS {
	case "linux":
		return parseProcMeminfo()
	case "windows":
		return getWindowsMemory()
	case "darwin", "freebsd":
		return getUnixMemory()
	default:
		return getGenericMemory()
	}
}

func parseProcMeminfo() (*VirtualMemoryStat, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	vm := &VirtualMemoryStat{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])
		valueStr = strings.TrimSuffix(valueStr, " kB")

		value, _ := strconv.ParseUint(valueStr, 10, 64)
		value *= 1024 // Convert from KB to bytes

		switch key {
		case "MemTotal":
			vm.Total = value
		case "MemFree":
			vm.Free = value
		case "MemAvailable":
			vm.Available = value
		case "Buffers":
			vm.Buffers = value
		case "Cached":
			vm.Cached = value
		case "Active":
			vm.Active = value
		case "Inactive":
			vm.Inactive = value
		case "Wired":
			vm.Wired = value
		case "Laundry":
			vm.Laundry = value
		case "Shmem":
			vm.Shared = value
		case "Slab":
			vm.Slab = value
		case "SReclaimable":
			vm.Sreclaimable = value
		case "SUnreclaim":
			vm.Sunreclaim = value
		case "SwapTotal":
			vm.SwapTotal = value
		case "SwapFree":
			vm.SwapFree = value
		}
	}

	if vm.Total > 0 {
		vm.Used = vm.Total - vm.Free - vm.Buffers - vm.Cached
		vm.UsedPercent = float64(vm.Used) / float64(vm.Total) * 100
	}

	return vm, scanner.Err()
}

func getWindowsMemory() (*VirtualMemoryStat, error) {
	output, err := runCommandWithTimeout("wmic", "OS", "get", "FreePhysicalMemory,TotalVisibleMemorySize", "/format:csv")
	if err != nil {
		return getGenericMemory()
	}

	lines := strings.Split(string(output), "\n")
	vm := &VirtualMemoryStat{}

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		free, _ := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
		total, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)

		vm.Free = free * 1024
		vm.Total = total * 1024
		vm.Used = vm.Total - vm.Free
		vm.UsedPercent = float64(vm.Used) / float64(vm.Total) * 100
	}

	return vm, nil
}

func getUnixMemory() (*VirtualMemoryStat, error) {
	output, err := runCommandWithTimeout("sysctl", "-a")
	if err != nil {
		return getGenericMemory()
	}

	vm := &VirtualMemoryStat{}
	lines := strings.Split(string(output), "\n")

	var pageSize, pageNum uint64

	for _, line := range lines {
		if strings.HasPrefix(line, "hw.memsize:") {
			vm.Total, _ = strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "hw.memsize:")), 10, 64)
		} else if strings.HasPrefix(line, "hw.pagesize:") {
			pageSize, _ = strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "hw.pagesize:")), 10, 64)
		} else if strings.HasPrefix(line, "vm.page_free_count:") {
			pageNum, _ = strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "vm.page_free_count:")), 10, 64)
		} else if strings.HasPrefix(line, "vm.page_active_count:") {
			active, _ := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "vm.page_active_count:")), 10, 64)
			vm.Active = active * pageSize
		} else if strings.HasPrefix(line, "vm.page_inactive_count:") {
			inactive, _ := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "vm.page_inactive_count:")), 10, 64)
			vm.Inactive = inactive * pageSize
		} else if strings.HasPrefix(line, "vm.page_wire_count:") {
			wired, _ := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "vm.page_wire_count:")), 10, 64)
			vm.Wired = wired * pageSize
		}
	}

	if pageSize > 0 {
		vm.Free = pageNum * pageSize
	}

	if vm.Total > 0 {
		vm.Used = vm.Total - vm.Free
		vm.UsedPercent = float64(vm.Used) / float64(vm.Total) * 100
	}

	return vm, nil
}

func getGenericMemory() (*VirtualMemoryStat, error) {
	return &VirtualMemoryStat{
		Total:       0,
		Free:        0,
		Used:        0,
		UsedPercent: 0,
	}, nil
}

func (vm *VirtualMemory) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Total", bytesToHuman(vm.TotalBytes), "").
		AddField("Available", bytesToHuman(vm.AvailableBytes), "").
		AddField("Used", bytesToHuman(vm.UsedBytes), "").
		AddField("UsedPercent", vm.UsedPercent, "%").
		AddField("Free", bytesToHuman(vm.FreeBytes), "").
		AddField("Active", bytesToHuman(vm.ActiveBytes), "").
		AddField("Inactive", bytesToHuman(vm.InactiveBytes), "").
		AddField("Wired", bytesToHuman(vm.WiredBytes), "").
		AddField("Buffers", bytesToHuman(vm.BuffersBytes), "").
		AddField("Cached", bytesToHuman(vm.CachedBytes), "").
		AddField("Shared", bytesToHuman(vm.SharedBytes), "").
		AddField("Slab", bytesToHuman(vm.SlabBytes), "").
		AddField("SwapTotal", bytesToHuman(vm.SwapTotalBytes), "").
		AddField("SwapFree", bytesToHuman(vm.SwapFreeBytes), "").
		AddField("SwapUsed", bytesToHuman(vm.SwapUsedBytes), "").
		Build()
}

type SwapDevice struct {
	Device      string `json:"device"`
	UsedBytes   uint64 `json:"usedBytes"`
	FreeBytes   uint64 `json:"freeBytes"`
	TotalBytes  uint64 `json:"totalBytes"`
}

func NewSwapDevices() ([]SwapDevice, error) {
	switch runtime.GOOS {
	case "linux":
		return parseProcSwaps()
	case "windows":
		return getWindowsSwap()
	case "darwin", "freebsd":
		return getUnixSwap()
	default:
		return []SwapDevice{}, nil
	}
}

func parseProcSwaps() ([]SwapDevice, error) {
	file, err := os.Open("/proc/swaps")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []SwapDevice
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 4 || fields[0] == "Filename" {
			continue
		}

		total, _ := strconv.ParseUint(fields[2], 10, 64)
		used, _ := strconv.ParseUint(fields[3], 10, 64)

		result = append(result, SwapDevice{
			Device:     fields[0],
			UsedBytes:  used * 1024,
			FreeBytes:  (total - used) * 1024,
			TotalBytes: total * 1024,
		})
	}

	return result, scanner.Err()
}

func getWindowsSwap() ([]SwapDevice, error) {
	output, err := runCommandWithTimeout("wmic", "pagefile", "get", "Name,CurrentUsage,AllocatedBaseSize", "/format:csv")
	if err != nil {
		return []SwapDevice{}, nil
	}

	var result []SwapDevice
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		used, _ := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
		total, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)

		result = append(result, SwapDevice{
			Device:     strings.TrimSpace(parts[0]),
			UsedBytes:  used * 1024 * 1024,
			FreeBytes:  (total - used) * 1024 * 1024,
			TotalBytes: total * 1024 * 1024,
		})
	}

	return result, nil
}

func getUnixSwap() ([]SwapDevice, error) {
	output, err := runCommandWithTimeout("swapinfo", "-k")
	if err != nil {
		output, err = runCommandWithTimeout("swapctl", "-l")
		if err != nil {
			return []SwapDevice{}, nil
		}
	}

	var result []SwapDevice
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		total, _ := strconv.ParseUint(fields[1], 10, 64)
		used, _ := strconv.ParseUint(fields[2], 10, 64)

		result = append(result, SwapDevice{
			Device:     fields[0],
			UsedBytes:  used * 1024,
			FreeBytes:  (total - used) * 1024,
			TotalBytes: total * 1024,
		})
	}

	return result, nil
}

func (s SwapDevice) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Device", s.Device, "").
		AddField("Used", bytesToHuman(s.UsedBytes), "").
		AddField("Free", bytesToHuman(s.FreeBytes), "").
		AddField("Total", bytesToHuman(s.TotalBytes), "").
		Build()
}

func SwapDevicesToPrint(swaps []SwapDevice) string {
	var sb strings.Builder
	for i, s := range swaps {
		sb.WriteString(s.ToPrint())
		if i < len(swaps)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
