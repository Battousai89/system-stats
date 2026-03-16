//go:build linux
// +build linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"system-stats/internal/types"
)

// GetVirtualMemory gets memory information on Linux
func GetVirtualMemory() (*types.VirtualMemory, error) {
	content, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}

	memInfo := parseMemInfo(content)

	total := memInfo["MemTotal"] * 1024     // Convert KB to bytes
	free := memInfo["MemFree"] * 1024
	available := memInfo["MemAvailable"] * 1024
	buffers := memInfo["Buffers"] * 1024
	cached := memInfo["Cached"] * 1024
	wired := memInfo["Slab"] * 1024
	committed := memInfo["Committed_AS"] * 1024
	commitLimit := memInfo["CommitLimit"] * 1024

	used := total - available
	if used < 0 {
		used = total - free
	}

	var percent float64
	if total > 0 {
		percent = float64(used) / float64(total) * 100.0
	}

	// Get page file size
	pageFile := memInfo["SwapTotal"] * 1024

	mem := &types.VirtualMemory{
		Total:       total,
		Available:   available,
		Used:        used,
		Free:        free,
		Percent:     percent,
		Active:      memInfo["Active"] * 1024,
		Inactive:    memInfo["Inactive"] * 1024,
		Cached:      cached,
		Buffers:     buffers,
		Wired:       wired,
		Committed:   committed,
		CommitLimit: commitLimit,
		PageFile:    pageFile,
	}

	return mem, nil
}

// GetSwapDevices gets swap device information on Linux
func GetSwapDevices() ([]types.SwapDevice, error) {
	content, err := os.ReadFile("/proc/swaps")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/swaps: %w", err)
	}

	return parseSwaps(content)
}

// parseMemInfo parses /proc/meminfo content
func parseMemInfo(content []byte) map[string]uint64 {
	result := make(map[string]uint64)

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])
		
		// Remove "kB" suffix if present
		valueStr = strings.TrimSpace(strings.TrimSuffix(valueStr, "kB"))
		
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			continue
		}

		result[key] = value
	}

	return result
}

// parseSwaps parses /proc/swaps content
func parseSwaps(content []byte) ([]types.SwapDevice, error) {
	var result []types.SwapDevice

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	
	// Skip header line
	if scanner.Scan() {
		// Header: Filename\t\tType\t\tSize\tUsed\tPriority
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Fields: Name, Type, Size, Used, Priority
		name := fields[0]
		size, _ := strconv.ParseUint(fields[2], 10, 64) // Size in KB
		used, _ := strconv.ParseUint(fields[3], 10, 64) // Used in KB

		// Convert KB to bytes
		total := size * 1024
		usedBytes := used * 1024
		freeBytes := total - usedBytes

		var percent float64
		if total > 0 {
			percent = float64(usedBytes) / float64(total) * 100.0
		}

		swap := types.SwapDevice{
			Name:        name,
			Total:       total,
			Used:        usedBytes,
			Free:        freeBytes,
			Percent:     percent,
			CurrentSize: usedBytes,
			PeakSize:    usedBytes, // Linux doesn't track peak usage per swap
		}

		result = append(result, swap)
	}

	if len(result) == 0 {
		// No swap devices - return empty slice, not an error
		return []types.SwapDevice{}, nil
	}

	return result, nil
}

// NewVirtualMemory creates VirtualMemory (compatibility function)
func NewVirtualMemory(vm map[string]any) *types.VirtualMemory {
	return nil
}

// NewSwapDevices creates SwapDevices list (compatibility function)
func NewSwapDevices() ([]types.SwapDevice, error) {
	return GetSwapDevices()
}

// MemoryStats holds all memory statistics
type MemoryStats struct {
	Virtual *types.VirtualMemory `json:"virtual"`
	Swap    []types.SwapDevice   `json:"swap"`
}

// GetAllMemoryStats collects all memory statistics
func GetAllMemoryStats() (*MemoryStats, error) {
	result := &MemoryStats{}

	vm, err := GetVirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory: %w", err)
	}
	result.Virtual = vm

	swap, err := GetSwapDevices()
	if err != nil {
		result.Swap = []types.SwapDevice{}
	} else {
		result.Swap = swap
	}

	return result, nil
}
