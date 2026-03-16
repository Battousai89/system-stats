//go:build linux
// +build linux

package services

import (
	"system-stats/internal/constants"
	"system-stats/internal/platform/linux"
	"system-stats/internal/types"
)

func getDefaultDiskPath() string {
	return constants.DefaultDiskPathLinux
}

func getHostInfo() (*types.HostInfo, error) {
	return linux.NewHostInfo()
}

func getLoadAvg() (*types.LoadAvg, error) {
	return linux.NewLoadAvg()
}

func getLoadMisc() (*types.LoadMisc, error) {
	return linux.NewLoadMisc()
}

func getVirtualization() (*types.VirtualizationInfo, error) {
	return linux.NewVirtualizationInfo()
}

func getCPUInfo() ([]types.CPUInfo, error) {
	return linux.NewCPUInfo()
}

func getCPUTimes() ([]types.CPUTimes, error) {
	return linux.NewCPUTimes()
}

func getCPUPercent() ([]types.CPUPercent, error) {
	return linux.NewCPUPercent()
}

func getVirtualMemory() (*types.VirtualMemory, error) {
	return linux.GetVirtualMemory()
}

func getSwapDevices() ([]types.SwapDevice, error) {
	return linux.GetSwapDevices()
}

func getDiskUsage(path string) (*types.DiskUsage, error) {
	return linux.NewDiskUsage(path)
}

func getDiskIOCounters() ([]types.DiskIOCounters, error) {
	return linux.NewDiskIOCounters()
}

func getAllDiskDeviceInfo() ([]types.DiskDeviceInfo, error) {
	return linux.GetAllDiskDeviceInfo()
}

func getNetIOCounters() ([]types.NetIOCounters, error) {
	return linux.NewNetIOCounters()
}

func getNetInterfaces() ([]types.NetInterface, error) {
	return linux.NewNetInterfaces()
}

func getNetProtocolCounters() ([]types.NetProtocolCounters, error) {
	return linux.NewNetProtocolCounters()
}

func getSensorTemperatures() ([]types.SensorTemperature, error) {
	return linux.NewSensorTemperatures()
}

func getBatteryInfo() ([]types.BatteryInfo, error) {
	return linux.NewBatteryInfo()
}

func getProcessInfo(topN int) ([]types.ProcessInfo, error) {
	return linux.NewProcessInfo(topN)
}

func getGPUInfo() ([]types.GPUInfo, error) {
	return linux.NewGPUInfo()
}

func getAllDockerStats() ([]types.DockerStats, error) {
	return linux.GetAllDockerStats()
}
