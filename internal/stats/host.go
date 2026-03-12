package stats

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"system-stats/internal/formatter"
)

type HostInfo struct {
	Hostname        string `json:"hostname"`
	UptimeSec       uint64 `json:"uptimeSec"`
	BootTimeUnix    uint64 `json:"bootTimeUnix"`
	Procs           uint64 `json:"procs"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformFamily  string `json:"platformFamily"`
	PlatformVersion string `json:"platformVersion"`
	KernelVersion   string `json:"kernelVersion"`
	KernelArch      string `json:"kernelArch"`
	HostID          string `json:"hostID"`
}

type LoadAvg struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type LoadMisc struct {
	ProcsTotal   uint64 `json:"procsTotal"`
	ProcsCreated uint64 `json:"procsCreated"`
	ProcsRunning uint64 `json:"procsRunning"`
	ProcsBlocked uint64 `json:"procsBlocked"`
	Ctxt         uint64 `json:"ctxt"`
}

type VirtualizationInfo struct {
	System string `json:"system"`
	Role   string `json:"role"`
}

type User struct {
	User     string `json:"user"`
	Terminal string `json:"terminal"`
	Host     string `json:"host"`
	Started  int    `json:"started"`
}

func NewHostInfo() (*HostInfo, error) {
	info := &HostInfo{
		OS:         runtime.GOOS,
		KernelArch: runtime.GOARCH,
	}

	hostname, err := os.Hostname()
	if err == nil {
		info.Hostname = hostname
	}

	switch runtime.GOOS {
	case "linux":
		info.getLinuxInfo()
	case "windows":
		info.getWindowsInfo()
	case "darwin":
		info.getDarwinInfo()
	case "freebsd":
		info.getFreeBSDInfo()
	default:
		info.Platform = runtime.GOOS
	}

	return info, nil
}

func (h *HostInfo) getLinuxInfo() {
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			uptime, _ := strconv.ParseFloat(fields[0], 64)
			h.UptimeSec = uint64(uptime)
		}
	}

	if data, err := os.ReadFile("/proc/stat"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "btime") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					h.BootTimeUnix, _ = strconv.ParseUint(fields[1], 10, 64)
				}
				break
			}
		}
	}

	if entries, err := os.ReadDir("/proc"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				if _, err := strconv.Atoi(entry.Name()); err == nil {
					h.Procs++
				}
			}
		}
	}

	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				h.Platform = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			} else if strings.HasPrefix(line, "ID=") {
				h.PlatformFamily = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
			} else if strings.HasPrefix(line, "VERSION_ID=") {
				h.PlatformVersion = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			}
		}
	}

	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		h.KernelVersion = strings.TrimSpace(string(data))
	}

	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		h.HostID = strings.TrimSpace(string(data))
	}
}

func (h *HostInfo) getWindowsInfo() {
	output, err := runCommandWithTimeout("wmic", "os", "get", "Caption,Version,NumberOfProcesses,LastBootUpTime", "/format:csv")
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue
			}
			parts := strings.Split(line, ",")
			if len(parts) >= 4 {
				h.Platform = strings.TrimSpace(parts[0])
				h.PlatformVersion = strings.TrimSpace(parts[1])
				h.Procs, _ = strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
			}
		}
	}

	output, err = runCommandWithTimeout("wmic", "os", "get", "Version", "/format:csv")
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue
			}
			h.KernelVersion = strings.TrimSpace(line)
			break
		}
	}
}

func (h *HostInfo) getDarwinInfo() {
	output, err := runCommandWithTimeout("sysctl", "-n", "kern.uptime")
	if err == nil {
		uptime, _ := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
		h.UptimeSec = uptime
	}

	output, err = runCommandWithTimeout("sw_vers", "-productName")
	if err == nil {
		h.Platform = strings.TrimSpace(string(output))
	}

	output, err = runCommandWithTimeout("sw_vers", "-productVersion")
	if err == nil {
		h.PlatformVersion = strings.TrimSpace(string(output))
	}

	output, err = runCommandWithTimeout("uname", "-r")
	if err == nil {
		h.KernelVersion = strings.TrimSpace(string(output))
	}

	output, err = runCommandWithTimeout("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "IOPlatformUUID") {
				parts := strings.Split(line, "=")
				if len(parts) > 1 {
					h.HostID = strings.TrimSpace(strings.Trim(parts[1], "\""))
				}
				break
			}
		}
	}
}

func (h *HostInfo) getFreeBSDInfo() {
	output, err := runCommandWithTimeout("sysctl", "-n", "kern.uptime")
	if err == nil {
		uptime, _ := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
		h.UptimeSec = uptime
	}

	output, err = runCommandWithTimeout("uname", "-s")
	if err == nil {
		h.Platform = strings.TrimSpace(string(output))
	}

	output, err = runCommandWithTimeout("uname", "-r")
	if err == nil {
		h.PlatformVersion = strings.TrimSpace(string(output))
		h.KernelVersion = strings.TrimSpace(string(output))
	}

	output, err = runCommandWithTimeout("ps", "-ax")
	if err == nil {
		lines := strings.Split(string(output), "\n")
		h.Procs = uint64(len(lines) - 1)
	}
}

func NewLoadAvg() (*LoadAvg, error) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		return parseProcLoadavg()
	case "darwin":
		return getDarwinLoadAvg()
	case "windows":
		return getWindowsLoadAvg()
	default:
		return &LoadAvg{}, nil
	}
}

func parseProcLoadavg() (*LoadAvg, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil, nil
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return &LoadAvg{
		Load1:  load1,
		Load5:  load5,
		Load15: load15,
	}, nil
}

func getDarwinLoadAvg() (*LoadAvg, error) {
	output, err := runCommandWithTimeout("sysctl", "-n", "vm.loadavg")
	if err != nil {
		return &LoadAvg{}, nil
	}

	outputStr := strings.Trim(string(output), "{} \n")
	fields := strings.Fields(outputStr)

	load := &LoadAvg{}
	if len(fields) >= 1 {
		load.Load1, _ = strconv.ParseFloat(fields[0], 64)
	}
	if len(fields) >= 2 {
		load.Load5, _ = strconv.ParseFloat(fields[1], 64)
	}
	if len(fields) >= 3 {
		load.Load15, _ = strconv.ParseFloat(fields[2], 64)
	}

	return load, nil
}

func getWindowsLoadAvg() (*LoadAvg, error) {
	output, err := runCommandWithTimeout("wmic", "cpu", "get", "LoadPercentage", "/format:csv")
	if err != nil {
		return &LoadAvg{}, nil
	}

	lines := strings.Split(string(output), "\n")
	var totalLoad float64
	var count int

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		load, _ := strconv.ParseFloat(strings.TrimSpace(line), 64)
		totalLoad += load
		count++
	}

	if count == 0 {
		return &LoadAvg{}, nil
	}

	avgLoad := totalLoad / float64(count) / 100

	return &LoadAvg{
		Load1:  avgLoad,
		Load5:  avgLoad,
		Load15: avgLoad,
	}, nil
}

func NewLoadMisc() (*LoadMisc, error) {
	misc := &LoadMisc{}

	switch runtime.GOOS {
	case "linux":
		if data, err := os.ReadFile("/proc/stat"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "processes") {
					fields := strings.Fields(line)
					if len(fields) > 1 {
						misc.ProcsCreated, _ = strconv.ParseUint(fields[1], 10, 64)
					}
				} else if strings.HasPrefix(line, "ctxt") {
					fields := strings.Fields(line)
					if len(fields) > 1 {
						misc.Ctxt, _ = strconv.ParseUint(fields[1], 10, 64)
					}
				}
			}
		}

		if entries, err := os.ReadDir("/proc"); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				if _, err := strconv.Atoi(entry.Name()); err == nil {
					misc.ProcsTotal++
					if data, err := os.ReadFile("/proc/" + entry.Name() + "/stat"); err == nil {
						fields := strings.Fields(string(data))
						if len(fields) > 2 {
							state := fields[2]
							if state == "R" {
								misc.ProcsRunning++
							} else if state == "D" {
								misc.ProcsBlocked++
							}
						}
					}
				}
			}
		}

	case "darwin", "freebsd":
		output, err := runCommandWithTimeout("ps", "-ax")
		if err == nil {
			lines := strings.Split(string(output), "\n")
			misc.ProcsTotal = uint64(len(lines) - 1)
			for _, line := range lines[1:] {
				if strings.Contains(line, " R ") {
					misc.ProcsRunning++
				}
			}
		}
	}

	return misc, nil
}

func NewVirtualizationInfo() (*VirtualizationInfo, error) {
	info := &VirtualizationInfo{}

	switch runtime.GOOS {
	case "linux":
		if data, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
			product := strings.ToLower(strings.TrimSpace(string(data)))
			if strings.Contains(product, "virtual") || strings.Contains(product, "vmware") || strings.Contains(product, "qemu") || strings.Contains(product, "kvm") {
				info.System = "kvm"
				info.Role = "guest"
				return info, nil
			}
		}

		if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			if strings.Contains(string(data), "hypervisor") {
				info.System = "hypervisor"
				info.Role = "guest"
				return info, nil
			}
		}

		if _, err := os.Stat("/.dockerenv"); err == nil {
			info.System = "docker"
			info.Role = "container"
			return info, nil
		}

		info.System = "native"
		info.Role = "host"

	case "windows":
		output, err := runCommandWithTimeout("wmic", "computersystem", "get", "model", "/format:csv")
		if err == nil {
			model := strings.ToLower(string(output))
			if strings.Contains(model, "virtual") || strings.Contains(model, "vmware") || strings.Contains(model, "hyper-v") {
				info.System = "hyperv"
				info.Role = "guest"
				return info, nil
			}
		}
		info.System = "native"
		info.Role = "host"

	case "darwin":
		output, err := runCommandWithTimeout("sysctl", "-a")
		if err == nil {
			if strings.Contains(string(output), "hypervisor") {
				info.System = "hypervisor"
				info.Role = "guest"
				return info, nil
			}
		}
		info.System = "native"
		info.Role = "host"

	default:
		info.System = "unknown"
		info.Role = "unknown"
	}

	return info, nil
}

func NewUsers() ([]User, error) {
	switch runtime.GOOS {
	case "linux", "freebsd":
		return parseUTMP()
	case "darwin":
		return getDarwinUsers()
	case "windows":
		return getWindowsUsers()
	default:
		return []User{}, nil
	}
}

func parseUTMP() ([]User, error) {
	output, err := runCommandWithTimeout("who")
	if err != nil {
		return []User{}, nil
	}

	var users []User
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			users = append(users, User{
				User:     fields[0],
				Terminal: fields[1],
				Host:     fields[2],
			})
		}
	}

	return users, nil
}

func getDarwinUsers() ([]User, error) {
	output, err := runCommandWithTimeout("who")
	if err != nil {
		return []User{}, nil
	}

	var users []User
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			users = append(users, User{
				User:     fields[0],
				Terminal: fields[1],
				Host:     fields[2],
			})
		}
	}

	return users, nil
}

func getWindowsUsers() ([]User, error) {
	output, err := runCommandWithTimeout("query", "user")
	if err != nil {
		return []User{}, nil
	}

	var users []User
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 1 {
			users = append(users, User{
				User:     fields[0],
				Terminal: "",
				Host:     "",
			})
		}
	}

	return users, nil
}

func (i HostInfo) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Hostname", i.Hostname, "").
		AddField("Uptime", formatDuration(i.UptimeSec), "").
		AddField("BootTime", i.BootTimeUnix, "unix").
		AddField("Procs", i.Procs, "").
		AddField("OS", i.OS, "").
		AddField("Platform", i.Platform, "").
		AddField("PlatformFamily", i.PlatformFamily, "").
		AddField("PlatformVersion", i.PlatformVersion, "").
		AddField("KernelVersion", i.KernelVersion, "").
		AddField("KernelArch", i.KernelArch, "").
		AddField("HostID", i.HostID, "").
		Build()
}

func (l LoadAvg) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Load1", l.Load1, "").
		AddField("Load5", l.Load5, "").
		AddField("Load15", l.Load15, "").
		Build()
}

func (l LoadMisc) ToPrint() string {
	return formatter.NewBuilder().
		AddField("ProcsTotal", l.ProcsTotal, "").
		AddField("ProcsCreated", l.ProcsCreated, "").
		AddField("ProcsRunning", l.ProcsRunning, "").
		AddField("ProcsBlocked", l.ProcsBlocked, "").
		AddField("Ctxt", l.Ctxt, "").
		Build()
}

func (v VirtualizationInfo) ToPrint() string {
	return formatter.NewBuilder().
		AddField("System", v.System, "").
		AddField("Role", v.Role, "").
		Build()
}

func (u User) ToPrint() string {
	return formatter.NewBuilder().
		AddField("User", u.User, "").
		AddField("Terminal", u.Terminal, "").
		AddField("Host", u.Host, "").
		AddField("Started", u.Started, "").
		Build()
}

func UsersToPrint(users []User) string {
	var sb strings.Builder
	for i, u := range users {
		sb.WriteString(u.ToPrint())
		if i < len(users)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
