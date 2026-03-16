//go:build linux
// +build linux

package linux

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// NewGPUInfo gets GPU information on Linux
func NewGPUInfo() ([]types.GPUInfo, error) {
	var result []types.GPUInfo

	// Try to get GPU info from multiple sources
	
	// Method 1: lspci
	if gpus, err := getGPUsFromLspci(); err == nil && len(gpus) > 0 {
		result = append(result, gpus...)
	}

	// Method 2: sysfs DRM interface
	if gpus, err := getGPUsFromDRM(); err == nil && len(gpus) > 0 {
		// Merge with existing GPUs
		result = mergeGPUs(result, gpus)
	}

	if len(result) == 0 {
		return []types.GPUInfo{}, nil
	}

	return result, nil
}

// gpuInfo holds intermediate GPU information
type gpuInfo struct {
	busID      string
	vendorID   string
	deviceID   string
	name       string
	driver     string
	memory     uint64
	memoryUsed uint64
	temp       float64
}

// getGPUsFromLspci gets GPU information using lspci
func getGPUsFromLspci() ([]types.GPUInfo, error) {
	// Use lspci to get VGA and 3D controllers
	output, err := helpers.RunCommandWithTimeout("lspci", "-nn", "-D")
	if err != nil {
		return nil, err
	}

	return parseLspciOutput(string(output))
}

// parseLspciOutput parses lspci output
func parseLspciOutput(output string) ([]types.GPUInfo, error) {
	var result []types.GPUInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if it's a GPU (VGA or 3D controller)
		if !strings.Contains(line, "VGA") && !strings.Contains(line, "3D") && !strings.Contains(line, "Display") {
			continue
		}

		// Parse the line
		// Format: 0000:00:02.0 VGA compatible controller [0300]: Intel Corporation Device [8086:3e9b] (rev 02)
		gpu := parseLspciLine(line)
		if gpu != nil {
			result = append(result, *gpu)
		}
	}

	return result, nil
}

// parseLspciLine parses a single lspci line
func parseLspciLine(line string) *types.GPUInfo {
	// Extract device ID from brackets
	deviceID := extractDeviceIDFromLspci(line)
	vendorID := extractVendorIDFromLspci(line)

	if deviceID == "" || vendorID == "" {
		return nil
	}

	// Get GPU name from device ID
	gpuName := types.GetGPUNameByDeviceID(deviceID)
	if gpuName == "" {
		// Try to extract name from the line
		gpuName = extractGPUNameFromLine(line)
	}

	// Determine manufacturer
	manufacturer := getGPUManufacturer(vendorID)

	gpu := &types.GPUInfo{
		Name:         gpuName,
		Manufacturer: manufacturer,
		DeviceID:     deviceID,
		VendorID:     vendorID,
	}

	// Try to get additional info from sysfs
	enrichGPUFromSysfs(gpu, vendorID, deviceID)

	return gpu
}

// extractDeviceIDFromLspci extracts device ID from lspci line
func extractDeviceIDFromLspci(line string) string {
	// Look for the last [vendor:device] pattern (e.g., [1002:98e4])
	lastBracket := strings.LastIndex(line, "[")
	if lastBracket == -1 {
		return ""
	}

	endBracket := strings.Index(line[lastBracket:], "]")
	if endBracket == -1 {
		return ""
	}

	idPart := line[lastBracket+1 : lastBracket+endBracket]
	parts := strings.Split(idPart, ":")
	if len(parts) != 2 {
		return ""
	}

	return "0x" + parts[1]
}

// extractVendorIDFromLspci extracts vendor ID from lspci line
func extractVendorIDFromLspci(line string) string {
	// Look for the last [vendor:device] pattern (e.g., [1002:98e4])
	lastBracket := strings.LastIndex(line, "[")
	if lastBracket == -1 {
		return ""
	}

	endBracket := strings.Index(line[lastBracket:], "]")
	if endBracket == -1 {
		return ""
	}

	idPart := line[lastBracket+1 : lastBracket+endBracket]
	parts := strings.Split(idPart, ":")
	if len(parts) != 2 {
		return ""
	}

	return "0x" + parts[0]
}

// extractGPUNameFromLine extracts GPU name from lspci line
func extractGPUNameFromLine(line string) string {
	// After the class description, before the vendor info
	// Format: ...: Vendor Name Device Name [id] ...
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return ""
	}

	// Find the bracket
	bracketIdx := strings.Index(line[colonIdx:], "[")
	if bracketIdx == -1 {
		return strings.TrimSpace(line[colonIdx+1:])
	}

	name := strings.TrimSpace(line[colonIdx+1 : colonIdx+bracketIdx])
	
	// Remove vendor prefix if present
	for _, vendor := range []string{"Intel", "AMD", "NVIDIA", "Advanced Micro Devices"} {
		name = strings.TrimPrefix(name, vendor+" ")
	}

	return name
}

// getGPUManufacturer determines manufacturer from vendor ID
func getGPUManufacturer(vendorID string) string {
	switch vendorID {
	case "0x10DE":
		return "NVIDIA"
	case "0x1002", "0x1022":
		return "AMD"
	case "0x8086":
		return "Intel"
	case "0x13B5":
		return "ARM"
	case "0x5143":
		return "Qualcomm"
	default:
		return "Unknown"
	}
}

// getGPUsFromDRM gets GPU information from DRM sysfs interface
func getGPUsFromDRM() ([]types.GPUInfo, error) {
	var result []types.GPUInfo

	// Look for DRM cards
	cards, err := filepath.Glob("/sys/class/drm/card*/device")
	if err != nil {
		return nil, err
	}

	for _, cardPath := range cards {
		gpu := types.GPUInfo{}

		// Read vendor ID
		if vendorID, err := os.ReadFile(filepath.Join(cardPath, "vendor")); err == nil {
			gpu.VendorID = normalizeVendorID(strings.TrimSpace(string(vendorID)))
		}

		// Read device ID
		if deviceID, err := os.ReadFile(filepath.Join(cardPath, "device")); err == nil {
			gpu.DeviceID = normalizeDeviceID(strings.TrimSpace(string(deviceID)))
		}

		// Get GPU name from IDs
		if gpu.DeviceID != "" {
			gpuName := types.GetGPUNameByDeviceID(gpu.DeviceID)
			if gpuName != "" {
				gpu.Name = gpuName
			}
		}

		// Get manufacturer
		if gpu.VendorID != "" {
			gpu.Manufacturer = getGPUManufacturer(gpu.VendorID)
		}

		// Try to get memory info
		gpu.Memory = getGPUMemory(cardPath)

		// Try to get driver info
		gpu.DriverVersion = getGPUDriver(cardPath)

		// Try to get temperature
		gpu.Temperature = getGPUTemperature(cardPath)

		if gpu.Name != "" || gpu.VendorID != "" {
			result = append(result, gpu)
		}
	}

	return result, nil
}

// normalizeVendorID normalizes vendor ID format
func normalizeVendorID(id string) string {
	id = strings.TrimPrefix(id, "0x")
	if len(id) >= 4 {
		return "0x" + id[len(id)-4:]
	}
	return id
}

// normalizeDeviceID normalizes device ID format
func normalizeDeviceID(id string) string {
	id = strings.TrimPrefix(id, "0x")
	if len(id) >= 4 {
		return "0x" + id[len(id)-4:]
	}
	return id
}

// getGPUMemory gets GPU memory from sysfs
func getGPUMemory(cardPath string) uint64 {
	// Try to read from hwmon
	memInfoPath := filepath.Join(cardPath, "mem_info_vram_total")
	if content, err := os.ReadFile(memInfoPath); err == nil {
		mem, _ := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
		return mem
	}

	// Try device-specific paths
	// For NVIDIA
	if content, err := os.ReadFile(filepath.Join(cardPath, "device", "mem_info_vram_total")); err == nil {
		mem, _ := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
		return mem
	}

	return 0
}

// getGPUDriver gets GPU driver version
func getGPUDriver(cardPath string) string {
	// Try to read driver name
	driverPath := filepath.Join(cardPath, "driver")
	if info, err := os.Readlink(driverPath); err == nil {
		return filepath.Base(info)
	}

	// Try modinfo for more details
	deviceName := filepath.Base(filepath.Dir(cardPath))
	if output, err := helpers.RunCommandWithTimeout("modinfo", "-F", "version", deviceName); err == nil {
		return strings.TrimSpace(string(output))
	}

	return ""
}

// getGPUTemperature gets GPU temperature
func getGPUTemperature(cardPath string) float64 {
	// Try hwmon
	hwmonPath := filepath.Join(cardPath, "hwmon")
	if files, err := os.ReadDir(hwmonPath); err == nil {
		for _, f := range files {
			if !f.IsDir() {
				continue
			}
			tempPath := filepath.Join(hwmonPath, f.Name(), "temp1_input")
			if content, err := os.ReadFile(tempPath); err == nil {
				temp, _ := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
				return temp / 1000.0 // Convert from millidegrees
			}
		}
	}

	return 0
}

// mergeGPUs merges GPU information from multiple sources
func mergeGPUs(existing, new []types.GPUInfo) []types.GPUInfo {
	existingMap := make(map[string]bool)
	for _, gpu := range existing {
		key := gpu.VendorID + ":" + gpu.DeviceID
		existingMap[key] = true
	}

	for _, gpu := range new {
		key := gpu.VendorID + ":" + gpu.DeviceID
		if !existingMap[key] {
			existing = append(existing, gpu)
		}
	}

	return existing
}

// enrichGPUFromSysfs enriches GPU information from sysfs
func enrichGPUFromSysfs(gpu *types.GPUInfo, vendorID, deviceID string) {
	// Try to find the GPU in sysfs
	drmPath := fmt.Sprintf("/sys/class/drm/card*/device")
	cards, _ := filepath.Glob(drmPath)

	for _, cardPath := range cards {
		// Check vendor ID
		if vid, err := os.ReadFile(filepath.Join(cardPath, "vendor")); err == nil {
			if normalizeVendorID(strings.TrimSpace(string(vid))) != vendorID {
				continue
			}
		}

		// Check device ID
		if did, err := os.ReadFile(filepath.Join(cardPath, "device")); err == nil {
			if normalizeDeviceID(strings.TrimSpace(string(did))) != deviceID {
				continue
			}
		}

		// Found matching GPU, get additional info
		if mem := getGPUMemory(cardPath); mem > 0 {
			gpu.Memory = mem
		}

		if driver := getGPUDriver(cardPath); driver != "" {
			gpu.DriverVersion = driver
		}

		if temp := getGPUTemperature(cardPath); temp > 0 {
			gpu.Temperature = temp
		}

		break
	}
}

// getNVIDIAGPUInfo gets additional NVIDIA GPU info using nvidia-smi if available
func getNVIDIAGPUInfo() ([]types.GPUInfo, error) {
	output, err := helpers.RunCommandWithTimeout("nvidia-smi", "-q", "-x")
	if err != nil {
		return nil, err
	}

	// Parse XML output (simplified)
	return parseNvidiaSMI(string(output))
}

// parseNvidiaSMI parses nvidia-smi XML output
func parseNvidiaSMI(output string) ([]types.GPUInfo, error) {
	// Simplified parsing - in production would use proper XML parser
	var result []types.GPUInfo

	// Look for GPU entries
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, "<product_name>") {
			gpu := types.GPUInfo{
				Manufacturer: "NVIDIA",
			}

			// Extract name
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "<product_name>"), "</product_name>"))
			gpu.Name = name

			// Look for memory info in subsequent lines
			for j := i; j < len(lines) && j < i+20; j++ {
				if strings.Contains(lines[j], "<total_memory>") {
					memStr := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(lines[j], "<total_memory>"), "</total_memory>"))
					if mem, err := strconv.ParseUint(memStr, 10, 64); err == nil {
						gpu.Memory = mem * 1024 * 1024 // Convert MB to bytes
					}
				}
				if strings.Contains(lines[j], "<gpu_temp>") {
					tempStr := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(lines[j], "<gpu_temp>"), "</gpu_temp>"))
					if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
						gpu.Temperature = temp
					}
				}
			}

			result = append(result, gpu)
		}
	}

	return result, nil
}

// GetAllGPUStats collects all GPU statistics
type AllGPUStats struct {
	GPUs []types.GPUInfo `json:"gpus"`
}

// GetAllGPUStats gets all GPU information
func GetAllGPUStats() (*AllGPUStats, error) {
	gpus, err := NewGPUInfo()
	if err != nil {
		return nil, err
	}

	return &AllGPUStats{
		GPUs: gpus,
	}, nil
}
