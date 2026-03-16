//go:build linux
// +build linux

package linux

import (
	"os"
	"strings"

	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// NewVirtualizationInfo gets virtualization information on Linux
func NewVirtualizationInfo() (*types.VirtualizationInfo, error) {
	info := &types.VirtualizationInfo{
		Platform:       "Linux",
		PlatformFamily: "Linux",
	}

	// Get architecture
	info.Architecture = getArchitecture()

	// Get boot time
	info.BootTime = getBootTime()

	// Get platform version
	info.PlatformVersion = getPlatformVersion()

	// Detect virtualization
	info.Hypervisor, info.Virtualized = detectVirtualization()

	// Detect container
	info.ContainerType = detectContainer()

	// Determine guest type
	if info.Virtualized {
		info.GuestType = detectGuestType(info.Hypervisor)
	}

	return info, nil
}

// detectVirtualization detects virtualization on Linux
func detectVirtualization() (string, bool) {
	// Method 1: Check /proc/cpuinfo for hypervisor flag
	if isVirtualizedFromCPU() {
		// Try to determine hypervisor type
		if hypervisor := getHypervisorFromDMI(); hypervisor != "" {
			return hypervisor, true
		}
		return "Unknown Hypervisor", true
	}

	// Method 2: Check DMI information
	if hypervisor := getHypervisorFromDMI(); hypervisor != "" {
		return hypervisor, true
	}

	// Method 3: Check systemd-detect-virt if available
	if hypervisor := detectVirt(); hypervisor != "" {
		return hypervisor, true
	}

	return "", false
}

// isVirtualizedFromCPU checks for hypervisor flag in /proc/cpuinfo
func isVirtualizedFromCPU() bool {
	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return false
	}

	// Look for hypervisor flag in features
	return strings.Contains(string(content), "hypervisor")
}

// getHypervisorFromDMI gets hypervisor type from DMI information
func getHypervisorFromDMI() string {
	// Check product name
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
		"microsoft":  "Hyper-V",
		"amazon ec2": "Amazon EC2",
		"google":     "Google Cloud",
		"digitalocean": "DigitalOcean",
		"openstack":  "OpenStack",
	}

	for indicator, vmType := range vmIndicators {
		if strings.Contains(productNameStr, indicator) {
			return vmType
		}
	}

	// Check product manufacturer
	manufacturer, err := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	if err == nil {
		manufacturerStr := strings.ToLower(strings.TrimSpace(string(manufacturer)))
		
		for indicator, vmType := range vmIndicators {
			if strings.Contains(manufacturerStr, indicator) {
				return vmType
			}
		}

		// Additional manufacturers
		if strings.Contains(manufacturerStr, "xen") {
			return "Xen"
		}
		if strings.Contains(manufacturerStr, "oracle") {
			return "VirtualBox"
		}
	}

	return ""
}

// detectVirt uses systemd-detect-virt if available
func detectVirt() string {
	output, err := helpers.RunCommandWithTimeout("systemd-detect-virt")
	if err != nil {
		return ""
	}

	virtType := strings.TrimSpace(string(output))
	
	switch virtType {
	case "kvm":
		return "KVM"
	case "vmware":
		return "VMware"
	case "virtualbox":
		return "VirtualBox"
	case "qemu":
		return "QEMU"
	case "xen":
		return "Xen"
	case "microsoft":
		return "Hyper-V"
	case "amazon":
		return "Amazon EC2"
	case "google":
		return "Google Cloud"
	case "docker":
		return "Docker"
	case "lxc":
		return "LXC"
	case "openvz":
		return "OpenVZ"
	case "none":
		return ""
	default:
		if virtType != "" {
			return virtType
		}
	}

	return ""
}

// detectContainer detects if running in a container
func detectContainer() string {
	// Check for Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}

	// Check for Docker via cgroup
	if content, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		cgroup := string(content)
		if strings.Contains(cgroup, "docker") || strings.Contains(cgroup, "kubepods") {
			return "docker"
		}
	}

	// Check for LXC
	if content, err := os.ReadFile("/sys/class/dmi/id/chassis_type"); err == nil {
		if strings.TrimSpace(string(content)) == "3" {
			// Could be LXC
			if _, err := os.Stat("/usr/lib/lxc"); err == nil {
				return "lxc"
			}
		}
	}

	// Check for OpenVZ/Virtuozzo
	if _, err := os.Stat("/proc/vz"); err == nil {
		return "openvz"
	}

	// Check container environment file
	if content, err := os.ReadFile("/run/.containerenv"); err == nil {
		containerEnv := string(content)
		if strings.Contains(containerEnv, "docker") {
			return "docker"
		}
		if strings.Contains(containerEnv, "podman") {
			return "podman"
		}
	}

	return ""
}

// detectGuestType determines guest OS type
func detectGuestType(hypervisor string) string {
	switch hypervisor {
	case "Hyper-V":
		return "Windows VM"
	case "VMware":
		return "VMware Guest"
	case "VirtualBox":
		return "VirtualBox Guest"
	case "KVM":
		return "KVM Guest"
	case "Xen":
		return "Xen Guest"
	case "QEMU":
		return "QEMU Guest"
	default:
		return "VM Guest"
	}
}

// getPlatformVersion gets the platform version
func getPlatformVersion() string {
	// Try /etc/os-release
	content, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VERSION_ID=") {
			version := strings.TrimPrefix(line, "VERSION_ID=")
			version = strings.Trim(version, "\"'")
			return version
		}
	}

	return ""
}

// GetSystemInfo gets general system information
func GetSystemInfo() (map[string]string, error) {
	result := make(map[string]string)
	result["GOOS"] = "linux"
	result["GOARCH"] = getArchitecture()
	result["NumCPU"] = getNumCPU()
	return result, nil
}

// getArchitecture gets system architecture
func getArchitecture() string {
	output, err := helpers.RunCommandWithTimeout("uname", "-m")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// getNumCPU gets number of CPUs
func getNumCPU() string {
	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "1"
	}

	count := 0
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "processor") {
			count++
		}
	}

	if count == 0 {
		return "1"
	}

	return string(rune(count))
}
