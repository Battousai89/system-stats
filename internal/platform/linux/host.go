//go:build linux
// +build linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// NewHostInfo gets host information on Linux
func NewHostInfo() (*types.HostInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	uptime, err := readUptime()
	if err != nil {
		uptime = 0
	}

	osInfo := getOSInfo()

	kernelVersion, err := getKernelVersion()
	if err != nil {
		kernelVersion = "unknown"
	}

	arch := getArchitecture()

	virtualization := getVirtualizationFromDMI()

	info := &types.HostInfo{
		Hostname:        hostname,
		Uptime:          uptime,
		OS:              osInfo["OS"],
		Platform:        "Linux",
		PlatformFamily:  osInfo["PlatformFamily"],
		PlatformVersion: osInfo["PlatformVersion"],
		KernelVersion:   kernelVersion,
		KernelArch:      arch,
		Virtualization:  virtualization,
		Role:            "Server", // Default for Linux
	}

	return info, nil
}

// NewLoadAvg gets load average on Linux
func NewLoadAvg() (*types.LoadAvg, error) {
	content, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return &types.LoadAvg{
			Load1:  0,
			Load5:  0,
			Load15: 0,
		}, nil
	}

	fields := strings.Fields(string(content))
	if len(fields) < 3 {
		return &types.LoadAvg{
			Load1:  0,
			Load5:  0,
			Load15: 0,
		}, nil
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return &types.LoadAvg{
		Load1:  load1,
		Load5:  load5,
		Load15: load15,
	}, nil
}

// getOSInfo reads OS information from /etc/os-release
func getOSInfo() map[string]string {
	result := map[string]string{
		"OS":              "Linux",
		"PlatformFamily":  "Linux",
		"PlatformVersion": "",
	}

	content, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return result
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			result["OS"] = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		} else if strings.HasPrefix(line, "ID=") {
			result["PlatformFamily"] = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		} else if strings.HasPrefix(line, "VERSION_ID=") {
			result["PlatformVersion"] = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}

	return result
}

// getKernelVersion gets kernel version using uname
func getKernelVersion() (string, error) {
	output, err := helpers.RunCommandWithTimeout("uname", "-r")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// readUptime reads uptime from /proc/uptime
func readUptime() (uint64, error) {
	content, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(content))
	if len(fields) < 1 {
		return 0, fmt.Errorf("invalid uptime format")
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	return uint64(uptime), nil
}

// getVirtualizationFromDMI detects virtualization from DMI info
func getVirtualizationFromDMI() string {
	// Check DMI product name for VM indicators
	productName, err := os.ReadFile("/sys/class/dmi/id/product_name")
	if err != nil {
		return ""
	}

	productNameStr := strings.ToLower(strings.TrimSpace(string(productName)))

	// Check for common VM indicators
	vmIndicators := map[string]string{
		"virtualbox": "VirtualBox",
		"vmware":     "VMware",
		"kvm":        "KVM",
		"qemu":       "QEMU",
		"hyperv":     "Hyper-V",
		"amazon ec2": "Amazon EC2",
		"google":     "Google Cloud",
		"microsoft":  "Hyper-V",
	}

	for indicator, vmType := range vmIndicators {
		if strings.Contains(productNameStr, indicator) {
			return vmType
		}
	}

	return ""
}
