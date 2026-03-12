package stats

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"system-stats/internal/formatter"
)

type GPUInfo struct {
	Index         int    `json:"index"`
	Name          string `json:"name"`
	Vendor        string `json:"vendor"`
	MemoryBytes   uint64 `json:"memoryBytes"`
	MemoryUsed    uint64 `json:"memoryUsedBytes"`
	MemoryFree    uint64 `json:"memoryFreeBytes"`
	Utilization   float32 `json:"utilizationPercent"`
	Temperature   int32   `json:"temperatureC"`
}

func NewGPUInfo() ([]GPUInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return getLinuxGPUInfo()
	case "windows":
		return getWindowsGPUInfo()
	case "darwin":
		return getDarwinGPUInfo()
	default:
		return []GPUInfo{}, nil
	}
}

func getLinuxGPUInfo() ([]GPUInfo, error) {
	var result []GPUInfo

	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,memory.total,memory.used,memory.free,utilization.gpu,temperature.gpu", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}

			parts := strings.Split(line, ",")
			if len(parts) < 7 {
				continue
			}

			gpu := GPUInfo{}
			gpu.Index, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
			gpu.Name = strings.TrimSpace(parts[1])
			gpu.Vendor = "NVIDIA"

			memTotal, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
			memUsed, _ := strconv.ParseUint(strings.TrimSpace(parts[3]), 10, 64)
			memFree, _ := strconv.ParseUint(strings.TrimSpace(parts[4]), 10, 64)
			util, _ := strconv.ParseFloat(strings.TrimSpace(parts[5]), 32)
			temp, _ := strconv.ParseInt(strings.TrimSpace(parts[6]), 10, 32)

			gpu.MemoryBytes = memTotal * 1024 * 1024
			gpu.MemoryUsed = memUsed * 1024 * 1024
			gpu.MemoryFree = memFree * 1024 * 1024
			gpu.Utilization = float32(util)
			gpu.Temperature = int32(temp)

			result = append(result, gpu)
		}

		if len(result) > 0 {
			return result, nil
		}
	}

	entries, err := os.ReadDir("/sys/class/drm")
	if err != nil {
		return []GPUInfo{}, fmt.Errorf("no GPU found")
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "card") || strings.Contains(name, "-") {
			continue
		}

		gpu := GPUInfo{
			Index: len(result),
		}

		devicePath := "/sys/class/drm/" + name + "/device"

		vendorID := ""
		if vendorData, err := os.ReadFile(devicePath + "/vendor"); err == nil {
			vendorID = strings.TrimSpace(string(vendorData))
			switch vendorID {
			case "0x10de":
				gpu.Vendor = "NVIDIA"
			case "0x1002":
				gpu.Vendor = "AMD"
			case "0x8086":
				gpu.Vendor = "Intel"
			case "0x14e4":
				gpu.Vendor = "Broadcom"
			case "0x13b5":
				gpu.Vendor = "ARM"
			default:
				gpu.Vendor = vendorID
			}
		}

		deviceID := ""
		if deviceData, err := os.ReadFile(devicePath + "/device"); err == nil {
			deviceID = strings.TrimSpace(string(deviceData))
		}

		gpuName := getGPUNameFromSysfs(devicePath, vendorID, deviceID)
		
		if gpuName != "" {
			gpu.Name = gpuName
		} else if deviceID != "" {
			gpu.Name = deviceID
		} else {
			gpu.Name = name
		}

		if gpu.Vendor == "AMD" || gpu.Vendor == "Intel" {
			gpu.MemoryBytes, gpu.MemoryUsed, gpu.MemoryFree = getGPUMemoryFromSysfs(devicePath)
		}

		if hwmonDirs, err := os.ReadDir(devicePath + "/hwmon"); err == nil {
			for _, hwmonDir := range hwmonDirs {
				tempPath := devicePath + "/hwmon/" + hwmonDir.Name() + "/temp1_input"
				if tempData, err := os.ReadFile(tempPath); err == nil {
					temp, _ := strconv.ParseInt(strings.TrimSpace(string(tempData)), 10, 64)
					gpu.Temperature = int32(temp / 1000)
					break
				}
			}
		}

		result = append(result, gpu)
	}

	if len(result) == 0 {
		return []GPUInfo{}, fmt.Errorf("no GPU found")
	}

	return result, nil
}

func getGPUNameFromSysfs(devicePath, vendorID, deviceID string) string {
	if modaliasData, err := os.ReadFile(devicePath + "/modalias"); err == nil {
		modalias := strings.TrimSpace(string(modaliasData))
		if strings.HasPrefix(modalias, "pci:v") {
			parts := strings.Split(modalias, "d")
			if len(parts) > 1 {
				name := lookupGPUName(vendorID, deviceID)
				if name != "" {
					return name
				}
			}
		}
	}

	if driverPath, err := os.Readlink(devicePath + "/driver"); err == nil {
		driverName := filepath.Base(driverPath)
		switch driverName {
		case "amdgpu":
			return lookupGPUName(vendorID, deviceID)
		case "i915":
			return "Intel Integrated Graphics"
		case "nouveau":
			return "NVIDIA GPU (nouveau)"
		case "nvidia":
			return "NVIDIA GPU"
		}
	}

	if subVendorData, err := os.ReadFile(devicePath + "/subsystem_vendor"); err == nil {
		subVendor := strings.TrimSpace(string(subVendorData))
		if subVendor != vendorID {
			name := lookupGPUName(vendorID, deviceID)
			if name != "" {
				return name
			}
		}
	}

	return lookupGPUName(vendorID, deviceID)
}

func lookupGPUName(vendorID, deviceID string) string {
	if vendorID == "" || deviceID == "" {
		return ""
	}

	vid := strings.ToLower(vendorID)
	did := strings.ToLower(deviceID)

	switch vid {
	case "0x1002":
		if name, ok := AMDGPUIDs[did]; ok {
			return name
		}
		return "AMD Radeon Graphics"
	case "0x8086":
		if name, ok := IntelGPUIDs[did]; ok {
			return name
		}
		return "Intel HD Graphics"
	case "0x10de":
		if name, ok := NVIDIAGPUIDs[did]; ok {
			return name
		}
		return "NVIDIA GeForce"
	}

	return ""
}

func getGPUMemoryFromSysfs(devicePath string) (total, used, free uint64) {
	memInfoPath := devicePath + "/mem_info_vram_total"
	if data, err := os.ReadFile(memInfoPath); err == nil {
		total, _ = strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	}

	memUsedPath := devicePath + "/mem_info_vram_used"
	if data, err := os.ReadFile(memUsedPath); err == nil {
		used, _ = strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	}

	if total > 0 && used > 0 {
		free = total - used
	} else if total > 0 {
		free = total
	}

	if total == 0 {
		gttTotalPath := devicePath + "/mem_info_gtt_total"
		if data, err := os.ReadFile(gttTotalPath); err == nil {
			total, _ = strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		}
		gttUsedPath := devicePath + "/mem_info_gtt_used"
		if data, err := os.ReadFile(gttUsedPath); err == nil {
			used, _ = strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		}
		if total > 0 && used > 0 {
			free = total - used
		}
	}

	return
}

func getWindowsGPUInfo() ([]GPUInfo, error) {
	var result []GPUInfo

	cmd := exec.Command("wmic", "path", "win32_VideoController", "get", "Index,Name,AdapterRAM,CurrentBitsPerPixel", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return []GPUInfo{}, fmt.Errorf("no GPU found")
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 4 {
			continue
		}

		gpu := GPUInfo{
			Index: i - 1,
		}

		gpu.Name = strings.TrimSpace(parts[2])
		gpu.MemoryBytes, _ = strconv.ParseUint(strings.TrimSpace(parts[3]), 10, 64)

		nameLower := strings.ToLower(gpu.Name)
		if strings.Contains(nameLower, "nvidia") {
			gpu.Vendor = "NVIDIA"
		} else if strings.Contains(nameLower, "amd") || strings.Contains(nameLower, "radeon") {
			gpu.Vendor = "AMD"
		} else if strings.Contains(nameLower, "intel") {
			gpu.Vendor = "Intel"
		}

		result = append(result, gpu)
	}

	if len(result) == 0 {
		return []GPUInfo{}, fmt.Errorf("no GPU found")
	}

	return result, nil
}

func getDarwinGPUInfo() ([]GPUInfo, error) {
	var result []GPUInfo

	cmd := exec.Command("system_profiler", "SPDisplaysDataType")
	output, err := cmd.Output()
	if err != nil {
		return []GPUInfo{}, fmt.Errorf("no GPU found")
	}

	lines := strings.Split(string(output), "\n")
	currentGPU := GPUInfo{}
	inGPU := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(line, "Graphics/Displays:") {
			continue
		}

		if !inGPU && strings.Contains(trimmed, "Chipset Model:") {
			inGPU = true
			currentGPU = GPUInfo{
				Index: len(result),
			}
		}

		if inGPU {
			if strings.Contains(trimmed, "Chipset Model:") {
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) > 1 {
					currentGPU.Name = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(trimmed, "Vendor:") {
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) > 1 {
					currentGPU.Vendor = strings.TrimSpace(parts[1])
				}
			} else if strings.Contains(trimmed, "Total Number of Frames:") {
				if currentGPU.Name != "" {
					result = append(result, currentGPU)
				}
				inGPU = false
			}
		}
	}

	if inGPU && currentGPU.Name != "" {
		result = append(result, currentGPU)
	}

	if len(result) == 0 {
		return []GPUInfo{}, fmt.Errorf("no GPU found")
	}

	return result, nil
}

func (g GPUInfo) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Index", g.Index, "").
		AddField("Name", g.Name, "").
		AddField("Vendor", g.Vendor, "").
		AddField("Memory", bytesToHuman(g.MemoryBytes), "").
		AddField("MemoryUsed", bytesToHuman(g.MemoryUsed), "").
		AddField("MemoryFree", bytesToHuman(g.MemoryFree), "").
		AddField("Utilization", g.Utilization, "%").
		AddField("Temperature", g.Temperature, "°C").
		Build()
}

func GPUInfosToPrint(gpus []GPUInfo) string {
	var sb strings.Builder
	for i, g := range gpus {
		sb.WriteString(fmt.Sprintf("  GPU[%d]:\n%s", i, g.ToPrint()))
		if i < len(gpus)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
