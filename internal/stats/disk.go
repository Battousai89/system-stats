package stats

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"system-stats/internal/formatter"
)

type DiskUsage struct {
	Path              string  `json:"path"`
	Fstype            string  `json:"fstype"`
	TotalBytes        uint64  `json:"totalBytes"`
	FreeBytes         uint64  `json:"freeBytes"`
	UsedBytes         uint64  `json:"usedBytes"`
	UsedPercent       float64 `json:"usedPercent"`
	InodesTotal       uint64  `json:"inodesTotal"`
	InodesUsed        uint64  `json:"inodesUsed"`
	InodesFree        uint64  `json:"inodesFree"`
	InodesUsedPercent float64 `json:"inodesUsedPercent"`
}

type DiskIOCounters struct {
	Name             string `json:"name"`
	ReadCount        uint64 `json:"readCount"`
	MergedReadCount  uint64 `json:"mergedReadCount"`
	WriteCount       uint64 `json:"writeCount"`
	MergedWriteCount uint64 `json:"mergedWriteCount"`
	ReadBytes        uint64 `json:"readBytes"`
	WriteBytes       uint64 `json:"writeBytes"`
	ReadTime         uint64 `json:"readTimeMs"`
	WriteTime        uint64 `json:"writeTimeMs"`
	IopsInProgress   uint64 `json:"iopsInProgress"`
	IoTime           uint64 `json:"ioTimeMs"`
	WeightedIO       uint64 `json:"weightedIO"`
}

type DiskPartition struct {
	Device     string `json:"device"`
	Mountpoint string `json:"mountpoint"`
	Fstype     string `json:"fstype"`
	Opts       string `json:"opts"`
}

func NewDiskUsage(path string) (*DiskUsage, error) {
	stat := &syscall.Statfs_t{}
	err := syscall.Statfs(path, stat)
	if err != nil {
		return nil, err
	}

	total := uint64(stat.Blocks) * uint64(stat.Bsize)
	free := uint64(stat.Bfree) * uint64(stat.Bsize)
	available := uint64(stat.Bavail) * uint64(stat.Bsize)
	used := total - free

	usage := &DiskUsage{
		Path:        path,
		TotalBytes:  total,
		FreeBytes:   available,
		UsedBytes:   used,
		InodesTotal: uint64(stat.Files),
		InodesFree:  uint64(stat.Ffree),
	}

	if usage.InodesTotal > 0 {
		usage.InodesUsed = usage.InodesTotal - usage.InodesFree
		usage.InodesUsedPercent = float64(usage.InodesUsed) / float64(usage.InodesTotal) * 100
	}

	if total > 0 {
		usage.UsedPercent = float64(used) / float64(total) * 100
	}

	usage.Fstype = getFSType(path)

	return usage, nil
}

func getFSType(path string) string {
	switch runtime.GOOS {
	case "linux":
		return getLinuxFSType(path)
	case "darwin":
		return getDarwinFSType(path)
	default:
		return "unknown"
	}
}

func getLinuxFSType(path string) string {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return "unknown"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == path {
			return fields[2]
		}
	}

	if path == "/" {
		file, err := os.Open("/proc/mounts")
		if err != nil {
			return "unknown"
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 2 && fields[1] == "/" {
				return fields[2]
			}
		}
	}

	return "unknown"
}

func getDarwinFSType(path string) string {
	output, err := runCommandWithTimeout("df", "-t", path)
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 1 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 2 {
			return fields[0]
		}
	}

	return "unknown"
}

func NewDiskIOCounters() ([]DiskIOCounters, error) {
	switch runtime.GOOS {
	case "linux":
		return parseProcDiskstats()
	case "windows":
		return getWindowsDiskIO()
	case "darwin", "freebsd":
		return getUnixDiskIO()
	default:
		return []DiskIOCounters{}, nil
	}
}

func parseProcDiskstats() ([]DiskIOCounters, error) {
	file, err := os.Open("/proc/diskstats")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []DiskIOCounters
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}

		name := fields[2]
		if len(name) > 0 && !strings.HasSuffix(name, "0") && name != "sda" && name != "sdb" && name != "nvme0n1" {
			if !strings.HasPrefix(name, "sd") && !strings.HasPrefix(name, "nvme") && !strings.HasPrefix(name, "vd") {
				continue
			}
		}

		readCount, _ := strconv.ParseUint(fields[3], 10, 64)
		mergedRead, _ := strconv.ParseUint(fields[4], 10, 64)
		writeCount, _ := strconv.ParseUint(fields[7], 10, 64)
		mergedWrite, _ := strconv.ParseUint(fields[8], 10, 64)
		readBytes, _ := strconv.ParseUint(fields[5], 10, 64)
		writeBytes, _ := strconv.ParseUint(fields[9], 10, 64)
		readTime, _ := strconv.ParseUint(fields[6], 10, 64)
		writeTime, _ := strconv.ParseUint(fields[10], 10, 64)
		iopsInProgress, _ := strconv.ParseUint(fields[11], 10, 64)
		ioTime, _ := strconv.ParseUint(fields[12], 10, 64)
		weightedIO, _ := strconv.ParseUint(fields[13], 10, 64)

		result = append(result, DiskIOCounters{
			Name:             name,
			ReadCount:        readCount,
			MergedReadCount:  mergedRead,
			WriteCount:       writeCount,
			MergedWriteCount: mergedWrite,
			ReadBytes:        readBytes * 512,
			WriteBytes:       writeBytes * 512,
			ReadTime:         readTime,
			WriteTime:        writeTime,
			IopsInProgress:   iopsInProgress,
			IoTime:           ioTime,
			WeightedIO:       weightedIO,
		})
	}

	return result, scanner.Err()
}

func getWindowsDiskIO() ([]DiskIOCounters, error) {
	output, err := runCommandWithTimeout("wmic", "diskdrive", "get", "Name,BytesReadPerSec,BytesWrittenPerSec", "/format:csv")
	if err != nil {
		return []DiskIOCounters{}, nil
	}

	var result []DiskIOCounters
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		readBytes, _ := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
		writeBytes, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)

		result = append(result, DiskIOCounters{
			Name:       strings.TrimSpace(parts[0]),
			ReadBytes:  readBytes,
			WriteBytes: writeBytes,
		})
	}

	return result, nil
}

func getUnixDiskIO() ([]DiskIOCounters, error) {
	output, err := runCommandWithTimeout("iostat", "-x", "1", "1")
	if err != nil {
		return []DiskIOCounters{}, nil
	}

	var result []DiskIOCounters
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		if fields[0] == "device" || fields[0] == "disk" {
			continue
		}

		readBytes, _ := strconv.ParseUint(fields[5], 10, 64)
		writeBytes, _ := strconv.ParseUint(fields[6], 10, 64)

		result = append(result, DiskIOCounters{
			Name:       fields[0],
			ReadBytes:  readBytes,
			WriteBytes: writeBytes,
		})
	}

	return result, nil
}

func NewDiskPartitions(all bool) ([]DiskPartition, error) {
	switch runtime.GOOS {
	case "linux":
		return parseProcMounts(all)
	case "windows":
		return getWindowsPartitions()
	case "darwin", "freebsd":
		return getUnixPartitions()
	default:
		return []DiskPartition{}, nil
	}
}

func parseProcMounts(all bool) ([]DiskPartition, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []DiskPartition
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}

		if !all && !strings.HasPrefix(fields[0], "/dev/") {
			continue
		}

		result = append(result, DiskPartition{
			Device:     fields[0],
			Mountpoint: fields[1],
			Fstype:     fields[2],
			Opts:       fields[3],
		})
	}

	return result, scanner.Err()
}

func getWindowsPartitions() ([]DiskPartition, error) {
	output, err := runCommandWithTimeout("wmic", "logicaldisk", "get", "DeviceID,FileSystem,Name", "/format:csv")
	if err != nil {
		return []DiskPartition{}, nil
	}

	var result []DiskPartition
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		result = append(result, DiskPartition{
			Device:     strings.TrimSpace(parts[0]),
			Mountpoint: strings.TrimSpace(parts[2]),
			Fstype:     strings.TrimSpace(parts[1]),
		})
	}

	return result, nil
}

func getUnixPartitions() ([]DiskPartition, error) {
	output, err := runCommandWithTimeout("df", "-T")
	if err != nil {
		return []DiskPartition{}, nil
	}

	var result []DiskPartition
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		result = append(result, DiskPartition{
			Device:     fields[0],
			Fstype:     fields[1],
			Mountpoint: fields[6],
		})
	}

	return result, nil
}

type DiskDeviceInfo struct {
	Name         string `json:"name"`
	SerialNumber string `json:"serialNumber"`
	Label        string `json:"label"`
}

func NewDiskDeviceInfo(name string) (*DiskDeviceInfo, error) {
	serial := getDiskSerial(name)
	label := getDiskLabel(name)

	return &DiskDeviceInfo{
		Name:         name,
		SerialNumber: serial,
		Label:        label,
	}, nil
}

func getDiskSerial(name string) string {
	switch runtime.GOOS {
	case "linux":
		paths := []string{
			"/sys/block/" + name + "/device/serial",
			"/sys/block/" + name + "/serial",
			"/sys/block/" + name + "/device/rev_id",
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err == nil {
				serial := strings.TrimSpace(string(data))
				if serial != "" && serial != "0" && serial != "unknown" {
					return serial
				}
			}
		}

		vendorPath := "/sys/block/" + name + "/device/vendor"
		modelPath := "/sys/block/" + name + "/device/model"
		if vendorData, err := os.ReadFile(vendorPath); err == nil {
			if modelData, err := os.ReadFile(modelPath); err == nil {
				vendor := strings.TrimSpace(string(vendorData))
				model := strings.TrimSpace(string(modelData))
				if vendor != "" && model != "" {
					return vendor + ":" + model
				}
			}
		}

		output, err := runCommandWithTimeout("hdparm", "-I", "/dev/"+name)
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Serial Number:") {
					serial := strings.TrimSpace(strings.TrimPrefix(line, "Serial Number:"))
					if serial != "" {
						return serial
					}
				}
			}
		}

		output, err = runCommandWithTimeout("smartctl", "-i", "/dev/"+name)
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Serial Number:") {
					serial := strings.TrimSpace(strings.TrimPrefix(line, "Serial Number:"))
					if serial != "" {
						return serial
					}
				}
			}
		}

	case "windows":
		output, err := runCommandWithTimeout("wmic", "diskdrive", "where", "name='"+name+"'", "get", "serialnumber", "/format:csv")
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				parts := strings.Split(line, ",")
				if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}
	return ""
}

func getDiskLabel(name string) string {
	switch runtime.GOOS {
	case "linux":
		dir, err := os.ReadDir("/dev/disk/by-label")
		if err == nil {
			for _, entry := range dir {
				labelName := entry.Name()
				linkPath := "/dev/disk/by-label/" + labelName
				target, err := os.Readlink(linkPath)
				if err == nil {
					if strings.HasSuffix(target, name) || strings.HasSuffix(target, "/"+name) {
						return labelName
					}
					absTarget, err := filepath.Abs(filepath.Dir(linkPath) + "/" + target)
					if err == nil {
						if strings.HasSuffix(absTarget, "/"+name) || absTarget == "/dev/"+name {
							return labelName
						}
					}
				}
			}
		}

		if data, err := os.ReadFile("/proc/mounts"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					device := fields[0]
					mountpoint := fields[1]
					if device == "/dev/"+name || strings.HasPrefix(device, "/dev/"+name) {
						if mountpoint != "/" {
							return mountpoint
						}
					}
				}
			}
		}

	case "windows":
		output, err := runCommandWithTimeout("wmic", "volume", "where", "deviceid='"+name+"'", "get", "label", "/format:csv")
		if err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 1 {
				parts := strings.Split(lines[1], ",")
				if len(parts) >= 2 {
					label := strings.TrimSpace(parts[1])
					if label != "" {
						return label
					}
				}
			}
		}
	}
	return ""
}

func GetAllDiskDeviceInfo() ([]DiskDeviceInfo, error) {
	var names []string

	switch runtime.GOOS {
	case "linux":
		entries, err := os.ReadDir("/sys/block")
		if err != nil {
			return []DiskDeviceInfo{}, nil
		}

		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "dm-") {
				continue
			}
			names = append(names, name)
		}

	case "windows":
		output, err := runCommandWithTimeout("wmic", "diskdrive", "get", "name", "/format:csv")
		if err != nil {
			return []DiskDeviceInfo{}, nil
		}

		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue
			}
			names = append(names, strings.TrimSpace(line))
		}

	case "darwin":
		output, err := runCommandWithTimeout("diskutil", "list", "physical")
		if err != nil {
			return []DiskDeviceInfo{}, nil
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "/dev/disk") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					names = append(names, strings.TrimPrefix(fields[0], "/dev/"))
				}
			}
		}
	}

	var result []DiskDeviceInfo
	for _, name := range names {
		info, err := NewDiskDeviceInfo(name)
		if err != nil {
			continue
		}
		result = append(result, *info)
	}

	return result, nil
}

func (u DiskUsage) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Path", u.Path, "").
		AddField("Fstype", u.Fstype, "").
		AddField("Total", bytesToHuman(u.TotalBytes), "").
		AddField("Free", bytesToHuman(u.FreeBytes), "").
		AddField("Used", bytesToHuman(u.UsedBytes), "").
		AddField("UsedPercent", u.UsedPercent, "%").
		AddField("InodesTotal", u.InodesTotal, "").
		AddField("InodesUsed", u.InodesUsed, "").
		AddField("InodesFree", u.InodesFree, "").
		AddField("InodesUsedPercent", u.InodesUsedPercent, "%").
		Build()
}

func (c DiskIOCounters) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Name", c.Name, "").
		AddField("ReadCount", c.ReadCount, "").
		AddField("WriteCount", c.WriteCount, "").
		AddField("ReadBytes", bytesToHuman(c.ReadBytes), "").
		AddField("WriteBytes", bytesToHuman(c.WriteBytes), "").
		AddField("ReadTime", c.ReadTime, "ms").
		AddField("WriteTime", c.WriteTime, "ms").
		AddField("IoTime", c.IoTime, "ms").
		Build()
}

func (p DiskPartition) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Device", p.Device, "").
		AddField("Mountpoint", p.Mountpoint, "").
		AddField("Fstype", p.Fstype, "").
		AddField("Opts", p.Opts, "").
		Build()
}

func (d DiskDeviceInfo) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Name", d.Name, "").
		AddField("SerialNumber", d.SerialNumber, "").
		AddField("Label", d.Label, "").
		Build()
}

func DiskDeviceInfosToPrint(devices []DiskDeviceInfo) string {
	var sb strings.Builder
	for i, d := range devices {
		sb.WriteString(d.ToPrint())
		if i < len(devices)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func DiskIOCountersToPrint(counters []DiskIOCounters) string {
	var sb strings.Builder
	for i, c := range counters {
		sb.WriteString(c.ToPrint())
		if i < len(counters)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func DiskPartitionsToPrint(partitions []DiskPartition) string {
	var sb strings.Builder
	for i, p := range partitions {
		sb.WriteString(p.ToPrint())
		if i < len(partitions)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
