package types

import (
	"fmt"
	"strings"

	"system-stats/internal/helpers"
)

// DiskUsage информация об использовании диска
type DiskUsage struct {
	Device      string  `json:"device"`       // Устройство/диск
	MountPoint  string  `json:"mount_point"`  // Точка монтирования
	FSType      string  `json:"fs_type"`      // Тип файловой системы
	Total       uint64  `json:"total"`        // Общий размер (байты)
	Used        uint64  `json:"used"`         // Использовано (байты)
	Free        uint64  `json:"free"`         // Свободно (байты)
	Percent     float64 `json:"percent"`      // Процент использования
	InodesTotal uint64  `json:"inodes_total"` // Всего инод
	InodesUsed  uint64  `json:"inodes_used"`  // Использовано инод
	InodesFree  uint64  `json:"inodes_free"`  // Свободно инод
}

// ToPrint форматирует DiskUsage для вывода
func (d *DiskUsage) ToPrint() string {
	b := helpers.NewBuilder()

	if d.Device != "" {
		b.AddField("Device", d.Device, "")
	}
	if d.MountPoint != "" {
		b.AddField("Mount Point", d.MountPoint, "")
	}
	if d.FSType != "" {
		b.AddField("File System", d.FSType, "")
	}

	b.AddFieldWithFormatter("Total", d.Total, "", formatBytes)
	b.AddFieldWithFormatter("Used", d.Used, "", formatBytes)
	b.AddFieldWithFormatter("Free", d.Free, "", formatBytes)
	b.AddField("Percent", fmt.Sprintf("%.2f", d.Percent), "%")

	if d.InodesTotal > 0 {
		b.AddField("Inodes Total", d.InodesTotal, "")
		b.AddField("Inodes Used", d.InodesUsed, "")
		b.AddField("Inodes Free", d.InodesFree, "")
	}

	return b.Build()
}

// DiskIOCounters счетчики дискового I/O
type DiskIOCounters struct {
	Name         string `json:"name"`          // Имя диска
	ReadCount    uint64 `json:"read_count"`    // Количество чтений
	WriteCount   uint64 `json:"write_count"`   // Количество записей
	ReadBytes    uint64 `json:"read_bytes"`    // Прочитано байт
	WriteBytes   uint64 `json:"write_bytes"`   // Записано байт
	ReadTime     uint64 `json:"read_time"`     // Время чтения (мс)
	WriteTime    uint64 `json:"write_time"`    // Время записи (мс)
	IoTime       uint64 `json:"io_time"`       // Общее время I/O (мс)
	BusyTime     uint64 `json:"busy_time"`     // Время занятости (мс)
	ReadBytesPerSec uint64 `json:"read_bps"`   // Чтение байт/сек
	WriteBytesPerSec uint64 `json:"write_bps"` // Запись байт/сек
}

// ToPrint форматирует DiskIOCounters для вывода
func (d *DiskIOCounters) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Name", d.Name, "")
	b.AddField("Read Count", d.ReadCount, "")
	b.AddField("Write Count", d.WriteCount, "")
	b.AddFieldWithFormatter("Read Bytes", d.ReadBytes, "", formatBytes)
	b.AddFieldWithFormatter("Write Bytes", d.WriteBytes, "", formatBytes)
	b.AddField("Read Time", d.ReadTime, "ms")
	b.AddField("Write Time", d.WriteTime, "ms")
	b.AddField("IO Time", d.IoTime, "ms")
	b.AddField("Busy Time", d.BusyTime, "ms")

	if d.ReadBytesPerSec > 0 {
		b.AddFieldWithFormatter("Read/sec", d.ReadBytesPerSec, "", formatBytes)
	}
	if d.WriteBytesPerSec > 0 {
		b.AddFieldWithFormatter("Write/sec", d.WriteBytesPerSec, "", formatBytes)
	}

	return b.Build()
}

// DiskIOCountersToPrint форматирует список DiskIOCounters
func DiskIOCountersToPrint(counters []DiskIOCounters) string {
	var sb strings.Builder
	for i, c := range counters {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("  Disk %d:\n", i+1))
		sb.WriteString(c.ToPrint())
	}
	return sb.String()
}

// DiskDeviceInfo информация о дисковом устройстве
type DiskDeviceInfo struct {
	Name        string `json:"name"`        // Имя устройства
	Model       string `json:"model"`       // Модель
	Serial      string `json:"serial"`      // Серийный номер
	Label       string `json:"label"`       // Метка тома
	FileSystem  string `json:"file_system"` // Файловая система
	Total       uint64 `json:"total"`       // Общий размер (байты)
	Free        uint64 `json:"free"`        // Свободно (байты)
	DeviceType  string `json:"device_type"` // Тип устройства
	MediaType   string `json:"media_type"`  // Тип носителя
	IsRemovable bool   `json:"removable"`   // Съемный
	IsSSD       bool   `json:"is_ssd"`      // SSD
}

// ToPrint форматирует DiskDeviceInfo для вывода
func (d *DiskDeviceInfo) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Name", d.Name, "")
	
	if d.Model != "" {
		b.AddField("Model", d.Model, "")
	}
	if d.Serial != "" {
		b.AddField("Serial", d.Serial, "")
	}
	if d.Label != "" {
		b.AddField("Label", d.Label, "")
	}
	if d.FileSystem != "" {
		b.AddField("File System", d.FileSystem, "")
	}
	if d.DeviceType != "" {
		b.AddField("Device Type", d.DeviceType, "")
	}
	if d.MediaType != "" {
		b.AddField("Media Type", d.MediaType, "")
	}

	b.AddFieldWithFormatter("Total", d.Total, "", formatBytes)
	b.AddFieldWithFormatter("Free", d.Free, "", formatBytes)

	if d.IsRemovable {
		b.AddField("Removable", "Yes", "")
	}
	if d.IsSSD {
		b.AddField("SSD", "Yes", "")
	}

	return b.Build()
}

// DiskDeviceInfosToPrint форматирует список DiskDeviceInfo
func DiskDeviceInfosToPrint(devices []DiskDeviceInfo) string {
	var sb strings.Builder
	for i, d := range devices {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("  Disk %d:\n", i+1))
		sb.WriteString(d.ToPrint())
	}
	return sb.String()
}
