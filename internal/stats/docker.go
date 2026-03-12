package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"system-stats/internal/formatter"
	"system-stats/internal/utils"
)

type DockerContainer struct {
	ContainerID string `json:"containerID"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	Status      string `json:"status"`
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage uint64 `json:"memoryUsage"`
	MemoryLimit uint64 `json:"memoryLimit"`
}

type DockerStats struct {
	ContainerID string  `json:"containerID"`
	CPUPercent  float64 `json:"cpuPercent"`
	MemoryUsage uint64  `json:"memoryUsage"`
	MemoryLimit uint64  `json:"memoryLimit"`
	MemoryPercent float64 `json:"memoryPercent"`
	NetInput    uint64  `json:"netInput"`
	NetOutput   uint64  `json:"netOutput"`
	BlockRead   uint64  `json:"blockRead"`
	BlockWrite  uint64  `json:"blockWrite"`
	PidsCurrent uint64  `json:"pidsCurrent"`
	PidsLimit   uint64  `json:"pidsLimit"`
}

type DockerContainerInfo struct {
	ID      string `json:"Id"`
	Names   []string `json:"Names"`
	Image   string `json:"Image"`
	State   string `json:"State"`
	Status  string `json:"Status"`
}

type DockerStatsResponse struct {
	Read      string `json:"read"`
	PidsStats struct {
		Current uint64 `json:"current"`
		Limit   uint64 `json:"limit"`
	} `json:"pids_stats"`
	MemoryStats struct {
		Usage    uint64 `json:"usage"`
		Limit    uint64 `json:"limit"`
		Stats    map[string]uint64 `json:"stats"`
	} `json:"memory_stats"`
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs  uint32 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Major uint64 `json:"major"`
			Minor uint64 `json:"minor"`
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
}

func NewDockerContainers() ([]DockerContainer, error) {
	switch runtime.GOOS {
	case "linux":
		return getLinuxDockerContainers()
	default:
		return getDockerContainersViaCLI()
	}
}

func getLinuxDockerContainers() ([]DockerContainer, error) {
	client, err := createDockerClient()
	if err == nil {
		containers, err := getContainersFromAPI(client)
		if err == nil && len(containers) > 0 {
			return containers, nil
		}
	}

	return getDockerContainersViaCLI()
}

func createDockerClient() (*http.Client, error) {
	socketPath := "/var/run/docker.sock"
	if _, err := os.Stat(socketPath); err != nil {
		return nil, err
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	return client, nil
}

func getContainersFromAPI(client *http.Client) ([]DockerContainer, error) {
	resp, err := client.Get("http://localhost/containers/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var containers []DockerContainerInfo
	if err := json.Unmarshal(body, &containers); err != nil {
		return nil, err
	}

	var result []DockerContainer
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		result = append(result, DockerContainer{
			ContainerID: c.ID[:12],
			Name:        name,
			Image:       c.Image,
			Status:      c.Status,
		})
	}

	return result, nil
}

func getDockerContainersViaCLI() ([]DockerContainer, error) {
	output, err := runCommandWithTimeout("docker", "ps", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}")
	if err != nil {
		return []DockerContainer{}, fmt.Errorf("docker not available or no containers running (permission denied)")
	}

	var result []DockerContainer
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}

		containerID := parts[0]
		if len(containerID) > 12 {
			containerID = containerID[:12]
		}

		result = append(result, DockerContainer{
			ContainerID: containerID,
			Name:        parts[1],
			Image:       parts[2],
			Status:      parts[3],
		})
	}

	return result, nil
}

func NewDockerStats(containerID string) (*DockerStats, error) {
	client, err := createDockerClient()
	if err == nil {
		stats, err := getDockerStatsFromAPI(client, containerID)
		if err == nil {
			return stats, nil
		}
	}

	return getDockerStatsFromCLI(containerID)
}

func getDockerStatsFromAPI(client *http.Client, containerID string) (*DockerStats, error) {
	resp, err := client.Get("http://localhost/containers/" + containerID + "/stats?stream=0")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var dockerStats DockerStatsResponse
	if err := json.Unmarshal(body, &dockerStats); err != nil {
		return nil, err
	}

	stats := &DockerStats{
		ContainerID:   containerID[:12],
		MemoryUsage:   utils.BytesToMB(dockerStats.MemoryStats.Usage),
		MemoryLimit:   utils.BytesToMB(dockerStats.MemoryStats.Limit),
		PidsCurrent:   dockerStats.PidsStats.Current,
		PidsLimit:     dockerStats.PidsStats.Limit,
	}

	if stats.MemoryLimit > 0 {
		stats.MemoryPercent = float64(dockerStats.MemoryStats.Usage) / float64(dockerStats.MemoryStats.Limit) * 100
	}

	cpuDelta := dockerStats.CPUStats.CPUUsage.TotalUsage - dockerStats.PreCPUStats.CPUUsage.TotalUsage
	systemDelta := dockerStats.CPUStats.SystemUsage - dockerStats.PreCPUStats.SystemUsage

	if cpuDelta > 0 && systemDelta > 0 {
		cpuPercent := float64(cpuDelta) / float64(systemDelta) * 100.0
		if dockerStats.CPUStats.OnlineCPUs > 0 {
			cpuPercent *= float64(dockerStats.CPUStats.OnlineCPUs)
		}
		stats.CPUPercent = cpuPercent
	}

	for _, net := range dockerStats.Networks {
		stats.NetInput += net.RxBytes
		stats.NetOutput += net.TxBytes
	}

	for _, ioStat := range dockerStats.BlkioStats.IoServiceBytesRecursive {
		if strings.ToLower(ioStat.Op) == "read" {
			stats.BlockRead += ioStat.Value
		} else if strings.ToLower(ioStat.Op) == "write" {
			stats.BlockWrite += ioStat.Value
		}
	}

	return stats, nil
}

func getDockerStatsFromCLI(containerID string) (*DockerStats, error) {
	output, err := runCommandWithTimeout("docker", "stats", "--no-stream", "--format",
		"{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}\t{{.PIDs}}",
		"--filter", "id="+containerID)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 6 {
			continue
		}

		stats := &DockerStats{
			ContainerID: containerID[:12],
		}

		cpuPct := strings.TrimSuffix(parts[0], "%")
		stats.CPUPercent, _ = strconv.ParseFloat(cpuPct, 64)

		memParts := strings.Split(parts[1], "/")
		if len(memParts) >= 2 {
			stats.MemoryUsage = parseMemoryValue(memParts[0])
			stats.MemoryLimit = parseMemoryValue(memParts[1])
		}

		memPct := strings.TrimSuffix(parts[2], "%")
		stats.MemoryPercent, _ = strconv.ParseFloat(memPct, 64)

		netParts := strings.Split(parts[3], "/")
		if len(netParts) >= 2 {
			stats.NetInput = parseMemoryValue(netParts[0])
			stats.NetOutput = parseMemoryValue(netParts[1])
		}

		blockParts := strings.Split(parts[4], "/")
		if len(blockParts) >= 2 {
			stats.BlockRead = parseMemoryValue(blockParts[0])
			stats.BlockWrite = parseMemoryValue(blockParts[1])
		}

		stats.PidsCurrent, _ = strconv.ParseUint(parts[5], 10, 64)

		return stats, nil
	}

	return nil, fmt.Errorf("no stats available")
}

func parseMemoryValue(s string) uint64 {
	s = strings.TrimSpace(s)
	multiplier := uint64(1)

	if strings.HasSuffix(s, "GiB") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GiB")
	} else if strings.HasSuffix(s, "MiB") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MiB")
	} else if strings.HasSuffix(s, "KiB") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "KiB")
	} else if strings.HasSuffix(s, "GB") {
		multiplier = 1000 * 1000 * 1000
		s = strings.TrimSuffix(s, "GB")
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1000 * 1000
		s = strings.TrimSuffix(s, "MB")
	} else if strings.HasSuffix(s, "kB") {
		multiplier = 1000
		s = strings.TrimSuffix(s, "kB")
	} else if strings.HasSuffix(s, "B") {
		s = strings.TrimSuffix(s, "B")
	}

	value, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return uint64(value * float64(multiplier))
}

const maxDockerConcurrency = 5

func GetAllDockerStats() ([]DockerStats, error) {
	containers, err := NewDockerContainers()
	if err != nil {
		return []DockerStats{}, err
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		sem     = make(chan struct{}, maxDockerConcurrency)
		result  []DockerStats
	)

	for _, c := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			stats, err := NewDockerStats(containerID)
			if err != nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			result = append(result, *stats)
		}(c.ContainerID)
	}

	wg.Wait()

	if len(result) == 0 {
		return []DockerStats{}, fmt.Errorf("no running containers found")
	}

	return result, nil
}

func (c DockerContainer) ToPrint() string {
	return formatter.NewBuilder().
		AddField("ContainerID", c.ContainerID, "").
		AddField("Name", c.Name, "").
		AddField("Image", c.Image, "").
		AddField("Status", c.Status, "").
		AddField("CPUUsage", c.CPUUsage, "%").
		AddField("MemoryUsage", c.MemoryUsage, "MB").
		AddField("MemoryLimit", c.MemoryLimit, "MB").
		Build()
}

func (s DockerStats) ToPrint() string {
	return formatter.NewBuilder().
		AddField("ContainerID", s.ContainerID, "").
		AddField("CPUPercent", s.CPUPercent, "%").
		AddField("MemoryUsage", s.MemoryUsage, "MB").
		AddField("MemoryLimit", s.MemoryLimit, "MB").
		AddField("MemoryPercent", s.MemoryPercent, "%").
		AddField("PidsCurrent", s.PidsCurrent, "").
		Build()
}

func DockerContainersToPrint(containers []DockerContainer) string {
	var sb strings.Builder
	for i, c := range containers {
		sb.WriteString(fmt.Sprintf("  [%d] ", i+1))
		sb.WriteString(c.ToPrint())
		if i < len(containers)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func DockerStatsToPrint(stats []DockerStats) string {
	var sb strings.Builder
	for i, s := range stats {
		sb.WriteString(fmt.Sprintf("  [%d] ", i+1))
		sb.WriteString(s.ToPrint())
		if i < len(stats)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
