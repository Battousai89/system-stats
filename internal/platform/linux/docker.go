//go:build linux
// +build linux

package linux

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"system-stats/internal/config"
	"system-stats/internal/types"
)

// dockerStatsRaw raw data from docker stats
type dockerStatsRaw struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
	CPUPerc    string `json:"CPUPerc"`
	MemUsage   string `json:"MemUsage"`
	MemPerc    string `json:"MemPerc"`
	NetIO      string `json:"NetIO"`
	BlockIO    string `json:"BlockIO"`
	PIDs       string `json:"PIDs"`
}

var (
	dockerPath        string
	dockerPathOnce    sync.Once
	dockerChecked     bool
	dockerAvailable   bool
	dockerCheckTime   time.Time
	dockerCheckTTL    = 5 * time.Second
	dockerCheckMu     sync.Mutex
)

// findDockerPath finds Docker CLI path
func findDockerPath() string {
	dockerPathOnce.Do(func() {
		// Try standard PATH lookup
		if path, err := exec.LookPath("docker"); err == nil {
			dockerPath = path
			dockerChecked = true
			return
		}

		// Docker not found
		dockerChecked = true
	})

	return dockerPath
}

// isDockerAvailable checks if Docker daemon is available (with caching)
func isDockerAvailable() bool {
	dockerExe := findDockerPath()
	if dockerExe == "" {
		return false
	}

	dockerCheckMu.Lock()
	defer dockerCheckMu.Unlock()

	// Check cache
	if dockerChecked && time.Since(dockerCheckTime) < dockerCheckTTL {
		return dockerAvailable
	}

	// Perform check
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dockerExe, "info")
	dockerAvailable = cmd.Run() == nil
	dockerChecked = true
	dockerCheckTime = time.Now()

	return dockerAvailable
}

// GetAllDockerStats gets statistics for all Docker containers
func GetAllDockerStats() ([]types.DockerStats, error) {
	// Check for Docker
	dockerExe := findDockerPath()
	if dockerExe == "" {
		return nil, fmt.Errorf("docker not found in PATH")
	}

	// Check Docker daemon availability
	if !isDockerAvailable() {
		return nil, fmt.Errorf("docker daemon is not running or not accessible")
	}

	// Get stats for all running containers
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout*2)
	defer cancel()

	cmd := exec.CommandContext(ctx, dockerExe, "stats", "--no-stream", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker stats: %w", err)
	}

	var rawStats []dockerStatsRaw
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var stat dockerStatsRaw
		if err := json.Unmarshal([]byte(line), &stat); err != nil {
			continue
		}
		rawStats = append(rawStats, stat)
	}

	// Return empty slice if no containers
	if len(rawStats) == 0 {
		return []types.DockerStats{}, nil
	}

	// Get container statuses
	statuses := getContainerStatusesBatch(rawStats, dockerExe)

	result := make([]types.DockerStats, 0, len(rawStats))
	for i, rs := range rawStats {
		cpuPerc := parseFloatPercent(rs.CPUPerc)
		memUsage, memLimit := parseMemUsage(rs.MemUsage)
		memPerc := parseFloatPercent(rs.MemPerc)
		pids := parseUint(rs.PIDs)

		status := statuses[i]
		if status == "" {
			status = "running"
		}

		stat := types.DockerStats{
			ContainerID:   rs.ID,
			Name:          rs.Name,
			CPU:           cpuPerc,
			Memory:        memUsage,
			MemoryLimit:   memLimit,
			MemoryPercent: memPerc,
			NetIO:         rs.NetIO,
			BlockIO:       rs.BlockIO,
			PIDs:          pids,
			Status:        status,
		}

		result = append(result, stat)
	}

	return result, nil
}

// getContainerStatusesBatch gets statuses for all containers in one call
func getContainerStatusesBatch(rawStats []dockerStatsRaw, dockerExe string) []string {
	statuses := make([]string, len(rawStats))

	// Collect all container IDs
	containerIDs := make([]string, 0, len(rawStats))
	for _, rs := range rawStats {
		containerIDs = append(containerIDs, rs.ID)
	}

	// Single docker inspect call for all containers
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout*2)
	defer cancel()

	args := append([]string{"inspect", "--format", "{{.Name}} {{.State.Status}}"}, containerIDs...)
	cmd := exec.CommandContext(ctx, dockerExe, args...)
	output, err := cmd.Output()
	if err != nil {
		// Fall back to parallel requests
		return getContainerStatusesParallel(rawStats, dockerExe)
	}

	// Parse output: each line is "<name> <status>"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) >= 2 {
			statuses[i] = parts[len(parts)-1]
		} else if len(parts) == 1 {
			statuses[i] = parts[0]
		} else {
			statuses[i] = "unknown"
		}
	}

	return statuses
}

// getContainerStatusesParallel gets container statuses in parallel
func getContainerStatusesParallel(rawStats []dockerStatsRaw, dockerExe string) []string {
	statuses := make([]string, len(rawStats))
	var wg sync.WaitGroup

	maxConcurrent := 5
	sem := make(chan struct{}, maxConcurrent)

	for i, rs := range rawStats {
		wg.Add(1)
		go func(idx int, containerID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			status := getContainerStatusSingle(containerID, dockerExe)
			statuses[idx] = status
		}(i, rs.ID)
	}

	wg.Wait()
	return statuses
}

// getContainerStatusSingle gets status for a single container
func getContainerStatusSingle(containerID, dockerExe string) string {
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dockerExe, "inspect", "--format", "{{.State.Status}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// parseFloatPercent parses percentage string
func parseFloatPercent(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")

	var result float64
	fmt.Sscanf(s, "%f", &result)
	return result
}

// parseMemUsage parses memory usage string
func parseMemUsage(s string) (uint64, uint64) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0
	}

	usage := parseSize(strings.TrimSpace(parts[0]))
	limit := parseSize(strings.TrimSpace(parts[1]))

	return usage, limit
}

// parseSize parses size string (e.g., "100MiB", "1GiB")
func parseSize(s string) uint64 {
	s = strings.TrimSpace(s)

	var num float64
	var unit string
	fmt.Sscanf(s, "%f%s", &num, &unit)

	unit = strings.ToUpper(strings.TrimSpace(unit))

	switch unit {
	case "B":
		return uint64(num)
	case "KIB", "KB":
		return uint64(num * 1024)
	case "MIB", "MB":
		return uint64(num * 1024 * 1024)
	case "GIB", "GB":
		return uint64(num * 1024 * 1024 * 1024)
	case "TIB", "TB":
		return uint64(num * 1024 * 1024 * 1024 * 1024)
	default:
		return uint64(num)
	}
}

// parseUint parses uint from string
func parseUint(s string) uint32 {
	var result uint32
	fmt.Sscanf(s, "%d", &result)
	return result
}
