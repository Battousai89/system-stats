package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"system-stats/internal/config"
	"system-stats/internal/types"
)

type Mode string

const (
	ModeAll       Mode = "all"
	ModeCPU       Mode = "cpu"
	ModeMem       Mode = "mem"
	ModeDisk      Mode = "disk"
	ModeNet       Mode = "net"
	ModeHost      Mode = "host"
	ModeSensor    Mode = "sensor"
	ModeBattery   Mode = "battery"
	ModeProcess   Mode = "proc"
	ModeGPU       Mode = "gpu"
	ModeDocker    Mode = "docker"
	ModeVirt      Mode = "virt"
	ModeSwap      Mode = "swap"
	ModeNetProto  Mode = "netproto"
	ModeDiskInfo  Mode = "diskinfo"
	ModeLoadMisc  Mode = "loadmisc"
)

var OutputFormat = "json"

type SystemStats struct {
	Mode             Mode
	HostInfo         *types.HostInfo
	LoadAvg          *types.LoadAvg
	LoadMisc         *types.LoadMisc
	Virtualization   *types.VirtualizationInfo
	CPUInfo          []types.CPUInfo
	CPUTimes         []types.CPUTimes
	CPUPercent       []types.CPUPercent
	Memory           *types.VirtualMemory
	SwapDevices      []types.SwapDevice
	DiskUsage        *types.DiskUsage
	DiskIO           []types.DiskIOCounters
	DiskDeviceInfo   []types.DiskDeviceInfo
	NetIO            []types.NetIOCounters
	NetIfaces        []types.NetInterface
	NetProtoCounters []types.NetProtocolCounters
	Sensors          []types.SensorTemperature
	Batteries        []types.BatteryInfo
	Processes        []types.ProcessInfo
	GPUs             []types.GPUInfo
	DockerStats      []types.DockerStats
	BenchmarkInfo    map[string]string `json:"benchmark,omitempty"`
	CollectedStats   []string          `json:"collectedStats,omitempty"`
}

func NewSystemStats(mode Mode) (*SystemStats, error) {
	s := &SystemStats{Mode: mode}

	switch mode {
	case ModeAll:
		s.collectAll()
	case ModeHost:
		s.collectHost()
	case ModeLoadMisc:
		s.collectLoadMisc()
	case ModeVirt:
		s.collectVirt()
	case ModeCPU:
		s.collectCPU()
	case ModeMem:
		s.collectMem()
	case ModeSwap:
		s.collectSwap()
	case ModeDisk:
		s.collectDisk()
	case ModeDiskInfo:
		s.collectDiskInfo()
	case ModeNet:
		s.collectNet()
	case ModeNetProto:
		s.collectNetProto()
	case ModeSensor:
		s.collectSensors()
	case ModeBattery:
		s.collectBattery()
	case ModeProcess:
		s.collectProcess()
	case ModeGPU:
		s.collectGPU()
	case ModeDocker:
		s.collectDocker()
	}

	return s, nil
}

func NewSystemStatsFromMap(modeMap map[Mode]bool) (*SystemStats, error) {
	s := &SystemStats{}

	if modeMap[ModeAll] {
		s.Mode = ModeAll
		s.collectAll()
		s.CollectedStats = []string{"all"}
		return s, nil
	}

	startTotal := time.Now()
	var wg sync.WaitGroup
	benchmarkTimes := make(map[string]time.Duration)
	var mu sync.Mutex
	var collectedStats []string
	var statsMu sync.Mutex

	addCollected := func(name string) {
		statsMu.Lock()
		defer statsMu.Unlock()
		collectedStats = append(collectedStats, name)
	}

	if modeMap[ModeHost] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("host")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectHost()
				mu.Lock()
				benchmarkTimes["host"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectHost()
			}
		}()
	}

	if modeMap[ModeLoadMisc] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("loadmisc")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectLoadMisc()
				mu.Lock()
				benchmarkTimes["loadmisc"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectLoadMisc()
			}
		}()
	}

	if modeMap[ModeVirt] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("virt")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectVirt()
				mu.Lock()
				benchmarkTimes["virt"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectVirt()
			}
		}()
	}

	if modeMap[ModeCPU] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("cpu")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectCPU()
				mu.Lock()
				benchmarkTimes["cpu"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectCPU()
			}
		}()
	}

	if modeMap[ModeMem] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("mem")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectMem()
				mu.Lock()
				benchmarkTimes["mem"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectMem()
			}
		}()
	}

	if modeMap[ModeSwap] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("swap")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectSwap()
				mu.Lock()
				benchmarkTimes["swap"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectSwap()
			}
		}()
	}

	if modeMap[ModeDisk] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("disk")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectDisk()
				mu.Lock()
				benchmarkTimes["disk"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectDisk()
			}
		}()
	}

	if modeMap[ModeDiskInfo] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("diskinfo")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectDiskInfo()
				mu.Lock()
				benchmarkTimes["diskinfo"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectDiskInfo()
			}
		}()
	}

	if modeMap[ModeNet] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("net")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectNet()
				mu.Lock()
				benchmarkTimes["net"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectNet()
			}
		}()
	}

	if modeMap[ModeNetProto] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("netproto")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectNetProto()
				mu.Lock()
				benchmarkTimes["netproto"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectNetProto()
			}
		}()
	}

	if modeMap[ModeSensor] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("sensors")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectSensors()
				mu.Lock()
				benchmarkTimes["sensors"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectSensors()
			}
		}()
	}

	if modeMap[ModeBattery] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("battery")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectBattery()
				mu.Lock()
				benchmarkTimes["battery"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectBattery()
			}
		}()
	}

	if modeMap[ModeProcess] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("process")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectProcess()
				mu.Lock()
				benchmarkTimes["process"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectProcess()
			}
		}()
	}

	if modeMap[ModeGPU] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("gpu")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectGPU()
				mu.Lock()
				benchmarkTimes["gpu"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectGPU()
			}
		}()
	}

	if modeMap[ModeDocker] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addCollected("docker")
			if config.BenchmarkMode {
				start := time.Now()
				s.collectDocker()
				mu.Lock()
				benchmarkTimes["docker"] = time.Since(start)
				mu.Unlock()
			} else {
				s.collectDocker()
			}
		}()
	}

	wg.Wait()

	if config.BenchmarkMode {
		s.BenchmarkInfo = make(map[string]string)
		for k, v := range benchmarkTimes {
			s.BenchmarkInfo[k] = v.String()
		}
		s.BenchmarkInfo["total"] = time.Since(startTotal).String()
	}

	s.CollectedStats = collectedStats
	s.Mode = ModeAll
	return s, nil
}

func (s *SystemStats) collectAll() {
	startTotal := time.Now()
	var wg sync.WaitGroup
	benchmarkTimes := make(map[string]time.Duration)
	var mu sync.Mutex

	collectWithTiming := func(name string, fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if config.BenchmarkMode {
				start := time.Now()
				fn()
				mu.Lock()
				benchmarkTimes[name] = time.Since(start)
				mu.Unlock()
			} else {
				fn()
			}
		}()
	}

	collectWithTiming("host", s.collectHost)
	collectWithTiming("loadmisc", s.collectLoadMisc)
	collectWithTiming("virt", s.collectVirt)
	collectWithTiming("cpu", s.collectCPU)
	collectWithTiming("mem", s.collectMem)
	collectWithTiming("swap", s.collectSwap)
	collectWithTiming("disk", s.collectDisk)
	collectWithTiming("diskinfo", s.collectDiskInfo)
	collectWithTiming("net", s.collectNet)
	collectWithTiming("netproto", s.collectNetProto)
	collectWithTiming("sensors", s.collectSensors)
	collectWithTiming("battery", s.collectBattery)
	collectWithTiming("process", s.collectProcess)
	collectWithTiming("gpu", s.collectGPU)
	collectWithTiming("docker", s.collectDocker)

	wg.Wait()

	if config.BenchmarkMode {
		s.BenchmarkInfo = make(map[string]string)
		for k, v := range benchmarkTimes {
			s.BenchmarkInfo[k] = v.String()
		}
		s.BenchmarkInfo["total"] = time.Since(startTotal).String()
	}
}

func (s *SystemStats) collectHost() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.HostInfo, _ = getHostInfo()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.LoadAvg, _ = getLoadAvg()
	}()

	wg.Wait()
}

func (s *SystemStats) collectLoadMisc() {
	s.LoadMisc, _ = getLoadMisc()
}

func (s *SystemStats) collectVirt() {
	s.Virtualization, _ = getVirtualization()
}

func (s *SystemStats) collectCPU() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.CPUInfo, _ = getCPUInfo()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.CPUTimes, _ = getCPUTimes()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.CPUPercent, _ = getCPUPercent()
	}()

	wg.Wait()
}

func (s *SystemStats) collectMem() {
	s.Memory, _ = getVirtualMemory()
}

func (s *SystemStats) collectSwap() {
	s.SwapDevices, _ = getSwapDevices()
}

func (s *SystemStats) collectDisk() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.DiskUsage, _ = getDiskUsage(getDefaultDiskPath())
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.DiskIO, _ = getDiskIOCounters()
	}()

	wg.Wait()
}

func (s *SystemStats) collectDiskInfo() {
	s.DiskDeviceInfo, _ = getAllDiskDeviceInfo()
}

func (s *SystemStats) collectNet() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.NetIO, _ = getNetIOCounters()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.NetIfaces, _ = getNetInterfaces()
	}()

	wg.Wait()
}

func (s *SystemStats) collectNetProto() {
	s.NetProtoCounters, _ = getNetProtocolCounters()
}

func (s *SystemStats) collectSensors() {
	s.Sensors, _ = getSensorTemperatures()
}

func (s *SystemStats) collectBattery() {
	s.Batteries, _ = getBatteryInfo()
}

func (s *SystemStats) collectProcess() {
	s.Processes, _ = getProcessInfo(config.TopProcessesCount)
}

func (s *SystemStats) collectGPU() {
	s.GPUs, _ = getGPUInfo()
}

func (s *SystemStats) collectDocker() {
	dockerStats, err := getAllDockerStats()
	if err != nil {
		// Docker may not be installed
		s.DockerStats = []types.DockerStats{}
		return
	}
	s.DockerStats = dockerStats
}

func (s *SystemStats) ToPrint() string {
	if OutputFormat == "json" {
		return s.toJSON()
	}
	return s.toPrintList()
}

func (s *SystemStats) toJSON() string {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal stats: %v"}`, err)
	}
	return string(data)
}

func (s *SystemStats) toPrintList() string {
	var sb strings.Builder

	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	sb.WriteString("                      SYSTEM STATS                         \n")
	sb.WriteString("═══════════════════════════════════════════════════════════\n\n")

	if s.HostInfo != nil {
		sb.WriteString("┌ HOST INFO\n")
		sb.WriteString(s.HostInfo.ToPrint() + "\n")
		if s.LoadAvg != nil {
			sb.WriteString("┌ LOAD AVERAGE\n")
			sb.WriteString(s.LoadAvg.ToPrint() + "\n")
		}
	}

	if s.LoadMisc != nil {
		sb.WriteString("┌ LOAD MISC\n")
		sb.WriteString(s.LoadMisc.ToPrint() + "\n")
	}

	if s.Virtualization != nil {
		sb.WriteString("┌ VIRTUALIZATION\n")
		sb.WriteString(s.Virtualization.ToPrint() + "\n")
	}

	if len(s.CPUInfo) > 0 {
		sb.WriteString("┌ CPU INFO\n")
		sb.WriteString(types.CPUInfosToPrint(s.CPUInfo) + "\n")
		if len(s.CPUTimes) > 0 {
			sb.WriteString("┌ CPU TIMES\n")
			sb.WriteString(types.CPUTimesToPrint(s.CPUTimes) + "\n")
		}
		if len(s.CPUPercent) > 0 {
			sb.WriteString("┌ CPU USAGE\n")
			sb.WriteString(types.CPUPercentsToPrint(s.CPUPercent) + "\n")
		}
	}

	if s.Memory != nil {
		sb.WriteString("┌ MEMORY\n")
		sb.WriteString(s.Memory.ToPrint() + "\n")
	}

	if len(s.SwapDevices) > 0 {
		sb.WriteString("┌ SWAP DEVICES\n")
		sb.WriteString(types.SwapDevicesToPrint(s.SwapDevices) + "\n")
	}

	if s.DiskUsage != nil {
		sb.WriteString("┌ DISK USAGE (/)\n")
		sb.WriteString(s.DiskUsage.ToPrint() + "\n")
	}
	if len(s.DiskIO) > 0 {
		sb.WriteString("┌ DISK I/O\n")
		sb.WriteString(types.DiskIOCountersToPrint(s.DiskIO) + "\n")
	}

	if len(s.DiskDeviceInfo) > 0 {
		sb.WriteString("┌ DISK DEVICE INFO\n")
		sb.WriteString(types.DiskDeviceInfosToPrint(s.DiskDeviceInfo) + "\n")
	}

	if len(s.NetIO) > 0 {
		sb.WriteString("┌ NETWORK I/O\n")
		sb.WriteString(types.NetIOCountersToPrint(s.NetIO) + "\n")
	}
	if len(s.NetIfaces) > 0 {
		sb.WriteString("┌ NETWORK INTERFACES\n")
		sb.WriteString(types.NetInterfacesToPrint(s.NetIfaces) + "\n")
	}

	if len(s.NetProtoCounters) > 0 {
		sb.WriteString("┌ NETWORK PROTOCOL COUNTERS\n")
		sb.WriteString(types.NetProtocolCountersToPrint(s.NetProtoCounters) + "\n")
	}

	if len(s.Sensors) > 0 {
		sb.WriteString("┌ TEMPERATURES\n")
		sb.WriteString(types.SensorTemperaturesToPrint(s.Sensors) + "\n")
	}

	if len(s.Batteries) > 0 {
		sb.WriteString("┌ BATTERY\n")
		sb.WriteString(types.BatteryInfosToPrint(s.Batteries) + "\n")
	}

	if len(s.Processes) > 0 {
		sb.WriteString("┌ TOP PROCESSES (by CPU)\n")
		sb.WriteString(types.ProcessInfosToPrint(s.Processes) + "\n")
	}

	if len(s.GPUs) > 0 {
		sb.WriteString("┌ GPU INFO\n")
		sb.WriteString(types.GPUInfosToPrint(s.GPUs) + "\n")
	}

	if len(s.DockerStats) > 0 {
		sb.WriteString("┌ DOCKER CONTAINERS\n")
		sb.WriteString(types.DockerStatsToPrint(s.DockerStats) + "\n")
	}

	sb.WriteString("═══════════════════════════════════════════════════════════\n")
	return sb.String()
}

// ============================================================================
// Заглушки функций для несуществующих пока типов
// ============================================================================

