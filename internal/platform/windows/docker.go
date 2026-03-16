package windows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"system-stats/internal/config"
	"system-stats/internal/types"
)

// dockerStatsRaw сырые данные из docker stats
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

// dockerInspectRaw сырые данные из docker inspect
type dockerInspectRaw struct {
	State struct {
		Status string `json:"Status"`
	} `json:"State"`
}

var (
	dockerPath        string
	dockerPathOnce    sync.Once
	dockerChecked     bool
	dockerAvailable   bool
	dockerCheckTime   time.Time
	dockerCheckTTL    = 5 * time.Second // Перепроверяем каждые 5 секунд
	dockerCheckMu     sync.Mutex
)

// findDockerPath находит путь к Docker CLI
func findDockerPath() string {
	dockerPathOnce.Do(func() {
		// Сначала пробуем стандартный поиск в PATH
		if path, err := exec.LookPath("docker"); err == nil {
			dockerPath = path
			dockerChecked = true
			return
		}

		// На Windows проверяем стандартные пути установки Docker Desktop
		if runtime.GOOS == "windows" {
			commonPaths := []string{
				`C:\Program Files\Docker\Docker\resources\bin\docker.exe`,
				`C:\Program Files (x86)\Docker\Docker\resources\bin\docker.exe`,
				`%LOCALAPPDATA%\Docker\bin\docker.exe`,
				`%PROGRAMFILES%\Docker\Docker\resources\bin\docker.exe`,
			}

			for _, p := range commonPaths {
				// Раскрываем переменные окружения
				expanded := expandEnv(p)
				if _, err := exec.LookPath(expanded); err == nil {
					dockerPath = expanded
					dockerChecked = true
					return
				}
			}
		}

		// Docker не найден
		dockerChecked = true
	})

	return dockerPath
}

// expandEnv раскрывает переменные окружения в пути
func expandEnv(path string) string {
	if strings.Contains(path, "%") {
		return strings.ReplaceAll(strings.ReplaceAll(path,
			"%LOCALAPPDATA%", getEnv("LOCALAPPDATA")),
			"%PROGRAMFILES%", getEnv("PROGRAMFILES"))
	}
	return path
}

// getEnv получает переменную окружения
func getEnv(name string) string {
	// Простая реализация для Windows путей
	switch name {
	case "LOCALAPPDATA":
		return filepath.Join(getHomeDir(), "AppData", "Local")
	case "PROGRAMFILES":
		return os.Getenv("PROGRAMFILES")
	default:
		return os.Getenv(name)
	}
}

// getHomeDir получает домашнюю директорию
func getHomeDir() string {
	if home := os.Getenv("USERPROFILE"); home != "" {
		return home
	}
	return `C:\Users\Default`
}

// isDockerAvailable проверяет доступность Docker daemon (с кэшированием)
func isDockerAvailable() bool {
	dockerExe := findDockerPath()
	if dockerExe == "" {
		return false
	}

	dockerCheckMu.Lock()
	defer dockerCheckMu.Unlock()

	// Проверяем кэш
	if dockerChecked && time.Since(dockerCheckTime) < dockerCheckTTL {
		return dockerAvailable
	}

	// Выполняем проверку
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, dockerExe, "info")
	dockerAvailable = cmd.Run() == nil
	dockerChecked = true
	dockerCheckTime = time.Now()

	return dockerAvailable
}

// GetAllDockerStats получает статистику всех Docker контейнеров
func GetAllDockerStats() ([]types.DockerStats, error) {
	// Проверяем наличие Docker
	dockerExe := findDockerPath()
	if dockerExe == "" {
		return nil, fmt.Errorf("docker not found in PATH or standard installation locations")
	}

	// Проверяем доступность Docker daemon
	if !isDockerAvailable() {
		return nil, fmt.Errorf("docker daemon is not running or not accessible")
	}

	// Получаем статистику всех running контейнеров с статусом
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout*2)
	defer cancel()

	// Используем формат с включением статуса контейнера
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

	// Если контейнеров нет, возвращаем пустой срез
	if len(rawStats) == 0 {
		return []types.DockerStats{}, nil
	}

	// Получаем статусы всех контейнеров одним запросом
	statuses := getContainerStatusesBatch(rawStats, dockerExe)

	result := make([]types.DockerStats, 0, len(rawStats))
	for i, rs := range rawStats {
		// Парсим CPU процент
		cpuPerc := parseFloatPercent(rs.CPUPerc)

		// Парсим память
		memUsage, memLimit := parseMemUsage(rs.MemUsage)
		memPerc := parseFloatPercent(rs.MemPerc)

		// Парсим PIDs
		pids := parseUint(rs.PIDs)

		// Получаем статус из пакетно собранных данных
		status := statuses[i]
		if status == "" {
			status = "running" // По умолчанию для stats
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

// getContainerStatusesBatch получает статусы всех контейнеров одним запросом
func getContainerStatusesBatch(rawStats []dockerStatsRaw, dockerExe string) []string {
	statuses := make([]string, len(rawStats))
	
	// Собираем все ID контейнеров
	containerIDs := make([]string, 0, len(rawStats))
	for _, rs := range rawStats {
		containerIDs = append(containerIDs, rs.ID)
	}

	// Один запрос docker inspect для всех контейнеров
	ctx, cancel := context.WithTimeout(context.Background(), config.CommandTimeout*2)
	defer cancel()

	args := append([]string{"inspect", "--format", "{{.Name}} {{.State.Status}}"}, containerIDs...)
	cmd := exec.CommandContext(ctx, dockerExe, args...)
	output, err := cmd.Output()
	if err != nil {
		// Если пакетный запрос не удался, пробуем по одному
		return getContainerStatusesParallel(rawStats, dockerExe)
	}

	// Парсим вывод: каждая строка это "<name> <status>"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) >= 2 {
			statuses[i] = parts[len(parts)-1] // Статус последнее слово
		} else if len(parts) == 1 {
			statuses[i] = parts[0]
		} else {
			statuses[i] = "unknown"
		}
	}

	return statuses
}

// getContainerStatusesParallel получает статусы контейнеров параллельно
func getContainerStatusesParallel(rawStats []dockerStatsRaw, dockerExe string) []string {
	statuses := make([]string, len(rawStats))
	var wg sync.WaitGroup

	// Ограничиваем количество одновременных запросов
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

// getContainerStatusSingle получает статус одного контейнера
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

// parseFloatPercent парсит строку процента (например "50.00%")
func parseFloatPercent(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	
	var result float64
	fmt.Sscanf(s, "%f", &result)
	return result
}

// parseMemUsage парсит использование памяти (например "100MiB / 1GiB")
func parseMemUsage(s string) (uint64, uint64) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0
	}

	usage := parseSize(strings.TrimSpace(parts[0]))
	limit := parseSize(strings.TrimSpace(parts[1]))

	return usage, limit
}

// parseSize парсит размер (например "100MiB", "1GiB")
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

// parseUint парсит uint из строки
func parseUint(s string) uint32 {
	var result uint32
	fmt.Sscanf(s, "%d", &result)
	return result
}
