//go:build linux
// +build linux

package linux

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

var (
	diskDeviceInfoCache     []types.DiskDeviceInfo
	diskDeviceInfoCacheTime time.Time
	diskCacheTTL            = 30 * time.Second
	diskDeviceInfoOnce      sync.Once
)

// NewDiskUsage gets disk usage information on Linux
func NewDiskUsage(path string) (*types.DiskUsage, error) {
	if path == "" {
		path = "/"
	}

	// Use statfs to get disk usage
	var stat syscallStatfs
	if err := syscallStatfsPath(path, &stat); err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	var percent float64
	if total > 0 {
		percent = float64(used) / float64(total) * 100.0
	}

	// Get filesystem type
	fsType := getFSType(path)

	// Get device name
	device := getDeviceForPath(path)

	usage := &types.DiskUsage{
		Device:     device,
		MountPoint: path,
		FSType:     fsType,
		Total:      total,
		Used:       used,
		Free:       available, // Use available instead of free for user perspective
		Percent:    percent,
	}

	return usage, nil
}

// NewDiskIOCounters gets disk I/O counters on Linux
func NewDiskIOCounters() ([]types.DiskIOCounters, error) {
	content, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/diskstats: %w", err)
	}

	return parseDiskStats(content)
}

// GetAllDiskDeviceInfo gets information about all disk devices
func GetAllDiskDeviceInfo() ([]types.DiskDeviceInfo, error) {
	// Check cache
	if diskDeviceInfoCache != nil && time.Since(diskDeviceInfoCacheTime) < diskCacheTTL {
		return diskDeviceInfoCache, nil
	}

	result, err := collectDiskDeviceInfo()
	if err != nil {
		return nil, err
	}

	// Cache the result
	diskDeviceInfoCache = result
	diskDeviceInfoCacheTime = time.Now()

	return result, nil
}

// collectDiskDeviceInfo collects disk device information
func collectDiskDeviceInfo() ([]types.DiskDeviceInfo, error) {
	var result []types.DiskDeviceInfo

	// Get list of block devices
	devices, err := filepath.Glob("/sys/block/*")
	if err != nil {
		return nil, fmt.Errorf("failed to list block devices: %w", err)
	}

	for _, devicePath := range devices {
		deviceName := filepath.Base(devicePath)

		// Skip virtual/loop devices
		if shouldSkipDevice(deviceName) {
			continue
		}

		info := types.DiskDeviceInfo{
			Name: deviceName,
		}

		// Read model
		if model, err := os.ReadFile(filepath.Join(devicePath, "device", "model")); err == nil {
			info.Model = strings.TrimSpace(string(model))
		}

		// Read serial number
		if serial, err := os.ReadFile(filepath.Join(devicePath, "serial")); err == nil {
			info.Serial = strings.TrimSpace(string(serial))
		}

		// Read size
		if sizeStr, err := os.ReadFile(filepath.Join(devicePath, "size")); err == nil {
			size, _ := strconv.ParseUint(strings.TrimSpace(string(sizeStr)), 10, 64)
			info.Total = size * 512 // Sectors to bytes
		}

		// Check if SSD (rotational = 0 means SSD)
		if rotational, err := os.ReadFile(filepath.Join(devicePath, "queue", "rotational")); err == nil {
			isRotational, _ := strconv.ParseBool(strings.TrimSpace(string(rotational)))
			info.IsSSD = !isRotational
		}

		// Check if removable
		if removable, err := os.ReadFile(filepath.Join(devicePath, "removable")); err == nil {
			info.IsRemovable, _ = strconv.ParseBool(strings.TrimSpace(string(removable)))
		}

		// Get device type
		info.DeviceType = getDeviceType(devicePath)

		// Try to get more info using lsblk
		enrichWithLsblk(&info)

		result = append(result, info)
	}

	return result, nil
}

// parseDiskStats parses /proc/diskstats
func parseDiskStats(content []byte) ([]types.DiskIOCounters, error) {
	var result []types.DiskIOCounters

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}

		deviceName := fields[2]

		// Skip partitions, only report physical disks
		if !isPhysicalDisk(deviceName) {
			continue
		}

		readCount, _ := strconv.ParseUint(fields[3], 10, 64)
		readSectors, _ := strconv.ParseUint(fields[5], 10, 64)
		readTime, _ := strconv.ParseUint(fields[6], 10, 64)
		writeCount, _ := strconv.ParseUint(fields[7], 10, 64)
		writeSectors, _ := strconv.ParseUint(fields[9], 10, 64)
		writeTime, _ := strconv.ParseUint(fields[10], 10, 64)
		ioTime, _ := strconv.ParseUint(fields[12], 10, 64)
		busyTime, _ := strconv.ParseUint(fields[13], 10, 64)

		counter := types.DiskIOCounters{
			Name:       deviceName,
			ReadCount:  readCount,
			WriteCount: writeCount,
			ReadBytes:  readSectors * 512,
			WriteBytes: writeSectors * 512,
			ReadTime:   readTime,
			WriteTime:  writeTime,
			IoTime:     ioTime,
			BusyTime:   busyTime,
		}

		result = append(result, counter)
	}

	return result, nil
}

// isPhysicalDisk checks if device is a physical disk (not partition)
func isPhysicalDisk(deviceName string) bool {
	// Physical disks typically don't have numbers in their names
	// or are nvme drives with specific pattern
	if strings.HasPrefix(deviceName, "sd") && !strings.ContainsAny(deviceName[2:], "0123456789") {
		return true
	}
	if strings.HasPrefix(deviceName, "hd") && !strings.ContainsAny(deviceName[2:], "0123456789") {
		return true
	}
	if strings.HasPrefix(deviceName, "nvme") && !strings.Contains(deviceName, "n") {
		return true
	}
	if strings.HasPrefix(deviceName, "vd") && !strings.ContainsAny(deviceName[2:], "0123456789") {
		return true
	}
	if strings.HasPrefix(deviceName, "xvd") && !strings.ContainsAny(deviceName[3:], "0123456789") {
		return true
	}
	return false
}

// shouldSkipDevice checks if device should be skipped
func shouldSkipDevice(deviceName string) bool {
	skipDevices := map[string]bool{
		"loop":       true,
		"ram":        true,
		"fd":         true,
		"sr":         true, // CD-ROM
		"dm":         true, // Device mapper
		"md":         true, // RAID
		"zram":       true,
		"nbd":        true,
	}

	for skip := range skipDevices {
		if strings.HasPrefix(deviceName, skip) {
			return true
		}
	}

	return false
}

// getDeviceType returns device type string
func getDeviceType(devicePath string) string {
	// Check if it's an NVMe drive
	if strings.Contains(devicePath, "nvme") {
		return "NVMe"
	}

	// Check if it's a virtual disk
	if _, err := os.Stat(filepath.Join(devicePath, "device", "vendor")); err == nil {
		vendor, _ := os.ReadFile(filepath.Join(devicePath, "device", "vendor"))
		vendorStr := strings.TrimSpace(string(vendor))
		if strings.Contains(strings.ToLower(vendorStr), "qemu") {
			return "VirtIO"
		}
		if strings.Contains(strings.ToLower(vendorStr), "vmware") {
			return "VMware Virtual"
		}
	}

	return "SATA"
}

// enrichWithLsblk enriches disk info using lsblk command
func enrichWithLsblk(info *types.DiskDeviceInfo) {
	output, err := helpers.RunCommandWithTimeout("lsblk", "-Jn", "-o", "NAME,MODEL,SERIAL,TYPE,FSTYPE,MOUNTPOINT", "/dev/"+info.Name)
	if err != nil {
		return
	}

	// Try to parse JSON output
	var lsblkOutput struct {
		BlockDevices []struct {
			Name       string `json:"name"`
			Model      string `json:"model"`
			Serial     string `json:"serial"`
			Type       string `json:"type"`
			FsType     string `json:"fstype"`
			Mountpoint string `json:"mountpoint"`
		} `json:"blockdevices"`
	}

	if err := json.Unmarshal(output, &lsblkOutput); err != nil {
		return
	}

	if len(lsblkOutput.BlockDevices) > 0 {
		device := lsblkOutput.BlockDevices[0]
		if device.Model != "" && info.Model == "" {
			info.Model = device.Model
		}
		if device.Serial != "" && info.Serial == "" {
			info.Serial = device.Serial
		}
		info.MediaType = device.Type
		info.FileSystem = device.FsType
	}
}

// getFSType gets filesystem type for a path
func getFSType(path string) string {
	content, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "unknown"
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 && fields[1] == path {
			return fields[2]
		}
	}

	return "unknown"
}

// getDeviceForPath gets device name for a mount path
func getDeviceForPath(path string) string {
	content, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "unknown"
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == path {
			device := fields[0]
			// Convert /dev/sda1 to sda1
			if strings.HasPrefix(device, "/dev/") {
				return device[5:]
			}
			return device
		}
	}

	return "unknown"
}

// syscallStatfsPath performs statfs syscall
func syscallStatfsPath(path string, stat *syscallStatfs) error {
	return statfs(path, stat)
}

// syscallStatfs is a wrapper around syscall.Statfs
type syscallStatfs struct {
	Blocks  uint64
	Bfree   uint64
	Bavail  uint64
	Bsize   int64
	Files   uint64
	Ffree   uint64
}

// statfs is implemented using syscall
func statfs(path string, stat *syscallStatfs) error {
	// Use Go's syscall package
	return doStatfs(path, stat)
}
