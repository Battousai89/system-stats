//go:build windows
// +build windows

package services

import (
	"system-stats/internal/constants"
	"system-stats/internal/platform/windows"
	"system-stats/internal/types"
)

func getDefaultDiskPath() string {
	return constants.DefaultDiskPathWindows
}

func getHostInfo() (*types.HostInfo, error) {
	return windows.NewHostInfo()
}

func getLoadAvg() (*types.LoadAvg, error) {
	return windows.NewLoadAvg()
}

func getLoadMisc() (*types.LoadMisc, error) {
	return windows.NewLoadMisc()
}

func getVirtualization() (*types.VirtualizationInfo, error) {
	return windows.NewVirtualizationInfo()
}

func getCPUInfo() ([]types.CPUInfo, error) {
	return windows.NewCPUInfo()
}

func getCPUTimes() ([]types.CPUTimes, error) {
	return windows.NewCPUTimes()
}

func getCPUPercent() ([]types.CPUPercent, error) {
	return windows.NewCPUPercent()
}

func getVirtualMemory() (*types.VirtualMemory, error) {
	return windows.GetVirtualMemory()
}

func getSwapDevices() ([]types.SwapDevice, error) {
	return windows.GetSwapDevices()
}

func getDiskUsage(path string) (*types.DiskUsage, error) {
	return windows.NewDiskUsage(path)
}

func getDiskIOCounters() ([]types.DiskIOCounters, error) {
	return windows.NewDiskIOCounters()
}

func getAllDiskDeviceInfo() ([]types.DiskDeviceInfo, error) {
	return windows.GetAllDiskDeviceInfo()
}

func getNetIOCounters() ([]types.NetIOCounters, error) {
	return windows.NewNetIOCounters()
}

func getNetInterfaces() ([]types.NetInterface, error) {
	return windows.NewNetInterfaces()
}

func getNetProtocolCounters() ([]types.NetProtocolCounters, error) {
	return windows.NewNetProtocolCounters()
}

func getSensorTemperatures() ([]types.SensorTemperature, error) {
	return windows.NewSensorTemperatures()
}

func getBatteryInfo() ([]types.BatteryInfo, error) {
	return windows.NewBatteryInfo()
}

func getProcessInfo(topN int) ([]types.ProcessInfo, error) {
	return windows.NewProcessInfo(topN)
}

func getGPUInfo() ([]types.GPUInfo, error) {
	return windows.NewGPUInfo()
}

func getAllDockerStats() ([]types.DockerStats, error) {
	return windows.GetAllDockerStats()
}
