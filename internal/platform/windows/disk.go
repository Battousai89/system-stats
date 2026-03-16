package windows

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"system-stats/internal/constants"
	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

var (
	diskDeviceInfoCache      []types.DiskDeviceInfo
	diskDeviceInfoCacheTime  time.Time
	diskCacheTTL             = 30 * time.Second // Кэшируем на 30 секунд
	diskDeviceInfoOnce       sync.Once
)

// win32LogicalDisk структура для парсинга вывода Win32_LogicalDisk
type win32LogicalDisk struct {
	DeviceID     string `json:"DeviceID"`     // Имя диска (C:, D:, etc.)
	VolumeName   string `json:"VolumeName"`   // Метка тома
	FileSystem   string `json:"FileSystem"`   // Файловая система
	Size         uint64 `json:"Size"`         // Общий размер (байты)
	FreeSpace    uint64 `json:"FreeSpace"`    // Свободное место (байты)
	DriveType    uint32 `json:"DriveType"`    // Тип диска
	ProviderName string `json:"ProviderName"` // Для сетевых дисков
}

// win32PerfDisk структура для парсинга вывода Win32_PerfFormattedData_PerfDisk_LogicalDisk
type win32PerfDisk struct {
	Name              string `json:"Name"`
	DiskReadBytesPerSec  uint64 `json:"DiskReadBytesPerSec"`
	DiskWriteBytesPerSec uint64 `json:"DiskWriteBytesPerSec"`
	DiskReadsPerSec      uint64 `json:"DiskReadsPerSec"`
	DiskWritesPerSec     uint64 `json:"DiskWritesPerSec"`
	PercentDiskTime      uint64 `json:"PercentDiskTime"`
	PercentIdleTime      uint64 `json:"PercentIdleTime"`
	AvgDiskQueueLength   uint64 `json:"AvgDiskQueueLength"`
}

// win32PerfDiskPhysical структура для физических дисков
type win32PerfDiskPhysical struct {
	Name              string `json:"Name"`
	DiskReadBytesPerSec  uint64 `json:"DiskReadBytesPerSec"`
	DiskWriteBytesPerSec uint64 `json:"DiskWriteBytesPerSec"`
	DiskReadsPerSec      uint64 `json:"DiskReadsPerSec"`
	DiskWritesPerSec     uint64 `json:"DiskWritesPerSec"`
	PercentDiskTime      uint64 `json:"PercentDiskTime"`
	AvgDiskQueueLength   uint64 `json:"AvgDiskQueueLength"`
}

// win32DiskDrive структура для парсинга вывода Win32_DiskDrive
type win32DiskDrive struct {
	DeviceID      string `json:"DeviceID"`      // \\.\PHYSICALDRIVE0
	Model         string `json:"Model"`         // Модель
	SerialNumber  string `json:"SerialNumber"`  // Серийный номер
	Size          uint64 `json:"Size"`          // Размер (байты)
	MediaType     string `json:"MediaType"`     // Тип носителя
	InterfaceType string `json:"InterfaceType"` // Интерфейс
	Partitions    uint32 `json:"Partitions"`    // Количество разделов
	BytesPerSector uint32 `json:"BytesPerSector"`
	SectorsPerTrack uint32 `json:"SectorsPerTrack"`
	TracksPerCylinder uint32 `json:"TracksPerCylinder"`
	Cylinders     uint32 `json:"Cylinders"`
	IsSSD         bool   `json:"IsSSD"` // Определяется по модели
}

// win32Partition структура для связи физических и логических дисков
type win32Partition struct {
	DiskIndex     uint32 `json:"DiskIndex"`
	DeviceID      string `json:"DeviceID"`
	StartingOffset uint64 `json:"StartingOffset"`
	Size          uint64 `json:"Size"`
	Index         uint32 `json:"Index"`
}

// NewDiskUsage получает информацию об использовании диска
func NewDiskUsage(path string) (*types.DiskUsage, error) {
	// Нормализуем путь к диску (C:\ -> C:)
	diskLetter := strings.TrimRight(path, "\\")
	if len(diskLetter) == 2 {
		diskLetter = diskLetter + ":"
	}
	diskLetter = strings.ToUpper(diskLetter[:1]) + ":"

	// Получаем все диски и фильтруем в Go
	script := `
		Get-CimInstance ` + constants.Win32LogicalDisk + ` `+
			`| Select-Object DeviceID,VolumeName,FileSystem,Size,FreeSpace,DriveType `+
			`| ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	var disks []win32LogicalDisk
	if err := helpers.ParseJSON(string(output), &disks); err != nil {
		var single win32LogicalDisk
		if err2 := helpers.ParseJSON(string(output), &single); err2 == nil {
			disks = []win32LogicalDisk{single}
		} else {
			return nil, fmt.Errorf("failed to parse disk usage JSON: %w", err)
		}
	}

	// Ищем нужный диск
	var disk win32LogicalDisk
	found := false
	for _, d := range disks {
		if strings.ToUpper(d.DeviceID) == diskLetter {
			disk = d
			found = true
			break
		}
	}

	if !found || disk.Size == 0 {
		return nil, fmt.Errorf("disk not found: %s", path)
	}

	used := disk.Size - disk.FreeSpace
	var percent float64
	if disk.Size > 0 {
		percent = float64(used) / float64(disk.Size) * 100.0
	}

	usage := &types.DiskUsage{
		Device:     disk.DeviceID,
		MountPoint: disk.DeviceID,
		FSType:     disk.FileSystem,
		Total:      disk.Size,
		Used:       used,
		Free:       disk.FreeSpace,
		Percent:    percent,
	}

	return usage, nil
}

// win32PerfDiskRaw структура для сырых данных о диске
type win32PerfDiskRaw struct {
	Name                    string  `json:"Name"`
	DiskReadBytesPerSec     float64 `json:"DiskReadBytesPerSec"`
	DiskWriteBytesPerSec    float64 `json:"DiskWriteBytesPerSec"`
	DiskReadBytesPerSecBase float64 `json:"DiskReadBytesPerSecBase"`
	DiskWriteBytesPerSecBase float64 `json:"DiskWriteBytesPerSecBase"`
	Timestamp_Sys           uint64  `json:"Timestamp_Sys"`
	Frequency_Sys           uint64  `json:"Frequency_Sys"`
}

// NewDiskIOCounters получает счетчики дискового I/O
func NewDiskIOCounters() ([]types.DiskIOCounters, error) {
	// Получаем сырые данные для кумулятивных счетчиков
	script := `
		Get-CimInstance Win32_PerfRawData_PerfDisk_LogicalDisk `+
			`| Where-Object { $_.Name -ne '_Total' } `+
			`| Select-Object Name,DiskReadBytesPerSec,DiskWriteBytesPerSec,`+
			`DiskReadBytesPerSecBase,DiskWriteBytesPerSecBase,`+
			`Timestamp_Sys,Frequency_Sys `+
			`| ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk IO counters: %w", err)
	}

	var perfDisks []win32PerfDiskRaw
	if err := helpers.ParseJSON(string(output), &perfDisks); err != nil {
		var single win32PerfDiskRaw
		if err2 := helpers.ParseJSON(string(output), &single); err2 == nil {
			perfDisks = []win32PerfDiskRaw{single}
		} else {
			return nil, fmt.Errorf("failed to parse disk IO JSON: %w", err)
		}
	}

	result := make([]types.DiskIOCounters, 0, len(perfDisks))
	for _, pd := range perfDisks {
		if pd.Name == "" || pd.Name == "_Total" {
			continue
		}

		// Конвертируем сырые значения в байты
		// Формула: RawValue / BaseValue для нормализации
		readBytes := uint64(pd.DiskReadBytesPerSec)
		writeBytes := uint64(pd.DiskWriteBytesPerSec)
		
		if pd.DiskReadBytesPerSecBase > 0 && pd.DiskReadBytesPerSecBase != 0xFFFFFFFF {
			readBytes = uint64(pd.DiskReadBytesPerSec / pd.DiskReadBytesPerSecBase * float64(pd.Frequency_Sys))
		}
		if pd.DiskWriteBytesPerSecBase > 0 && pd.DiskWriteBytesPerSecBase != 0xFFFFFFFF {
			writeBytes = uint64(pd.DiskWriteBytesPerSec / pd.DiskWriteBytesPerSecBase * float64(pd.Frequency_Sys))
		}

		counter := types.DiskIOCounters{
			Name:         pd.Name,
			ReadBytes:    readBytes,
			WriteBytes:   writeBytes,
			ReadCount:    readBytes / 65536, // Примерное количество операций (средний размер 64KB)
			WriteCount:   writeBytes / 65536,
		}

		result = append(result, counter)
	}

	return result, nil
}

// GetAllDiskDeviceInfo получает информацию о всех дисковых устройствах (с кэшированием)
func GetAllDiskDeviceInfo() ([]types.DiskDeviceInfo, error) {
	// Проверяем кэш
	if diskDeviceInfoCache != nil && time.Since(diskDeviceInfoCacheTime) < diskCacheTTL {
		return diskDeviceInfoCache, nil
	}

	// Получаем информацию о физических дисках через Win32_DiskDrive
	script := `
		Get-CimInstance ` + constants.Win32DiskDrive + ` | `+
			`Select-Object DeviceID,Model,SerialNumber,Size,MediaType,`+
			`InterfaceType,Partitions | `+
			`ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk drive info: %w", err)
	}

	var drives []win32DiskDrive
	if err := helpers.ParseJSON(string(output), &drives); err != nil {
		var single win32DiskDrive
		if err2 := helpers.ParseJSON(string(output), &single); err2 == nil {
			drives = []win32DiskDrive{single}
		} else {
			return nil, fmt.Errorf("failed to parse disk drive JSON: %w", err)
		}
	}

	// Пытаемся получить информацию о типах дисков через Get-PhysicalDisk (требует админ прав)
	physicalDisks := make(map[string]physicalDiskInfo)
	physicalScript := `
		Get-PhysicalDisk 2>$null | `+
			`Select-Object DeviceId,MediaType,RotationRate | `+
			`ConvertTo-Json`
	physicalOutput, physicalErr := helpers.RunPowerShellCommand(physicalScript)
	if physicalErr == nil {
		var pdList []physicalDiskInfo
		if err := helpers.ParseJSON(string(physicalOutput), &pdList); err == nil {
			for _, pd := range pdList {
				key := fmt.Sprintf("\\\\.\\PHYSICALDRIVE%d", pd.DeviceID)
				physicalDisks[key] = pd
			}
		}
	}

	resultDisks := make([]types.DiskDeviceInfo, 0, len(drives))
	for _, drive := range drives {
		// Определяем SSD через MSFT_PhysicalDisk или RotationRate
		isSSD := false
		if pd, ok := physicalDisks[drive.DeviceID]; ok {
			// MediaType: 0=Unknown, 3=HDD, 4=SSD, 5=SCM
			isSSD = (pd.MediaType == 4 || pd.MediaType == 5)
			// Если MediaType неизвестен (0), пробуем RotationRate
			if pd.MediaType == 0 && pd.RotationRate == 0 {
				isSSD = true
			}
		} else {
			// Fallback: эвристика по названию модели
			isSSD = isSSDDrive(drive.Model)
		}

		device := types.DiskDeviceInfo{
			Name:        drive.DeviceID,
			Model:       drive.Model,
			Serial:      drive.SerialNumber,
			Total:       drive.Size,
			DeviceType:  drive.InterfaceType,
			MediaType:   drive.MediaType,
			IsRemovable: drive.MediaType == "Removable" || drive.MediaType == "External",
			IsSSD:       isSSD,
		}

		resultDisks = append(resultDisks, device)
	}

	// Кэшируем результат
	diskDeviceInfoCache = resultDisks
	diskDeviceInfoCacheTime = time.Now()

	return resultDisks, nil
}

// physicalDiskInfo информация из MSFT_PhysicalDisk
type physicalDiskInfo struct {
	DeviceID     uint32 `json:"DeviceId"`
	MediaType    uint32 `json:"MediaType"`    // 0=Unknown, 3=HDD, 4=SSD, 5=SCM
	RotationRate uint32 `json:"RotationRate"` // 0=SSD, >0=RPM для HDD
}

// isSSDDrive определяет, является ли диск SSD по модели
func isSSDDrive(model string) bool {
	model = strings.ToLower(model)
	
	for _, indicator := range constants.SSDIndicators {
		if strings.Contains(model, indicator) {
			return true
		}
	}

	return false
}

// GetDiskUsage получает информацию об использовании диска (альтернативная функция)
func GetDiskUsage(path string) (*types.DiskUsage, error) {
	return NewDiskUsage(path)
}

// GetDiskIOCounters получает счетчики дискового I/O (альтернативная функция)
func GetDiskIOCounters() ([]types.DiskIOCounters, error) {
	return NewDiskIOCounters()
}

// GetAllDiskStats возвращает всю информацию о дисках
type AllDiskStats struct {
	Usage       []types.DiskUsage       `json:"usage"`
	IOCounters  []types.DiskIOCounters  `json:"io_counters"`
	DeviceInfo  []types.DiskDeviceInfo  `json:"device_info"`
}

// GetAllDiskStats собирает всю информацию о дисках
func GetAllDiskStats() (*AllDiskStats, error) {
	result := &AllDiskStats{}

	// Получаем логические диски
	logicalScript := `
		Get-CimInstance ` + constants.Win32LogicalDisk + ` `+
			`| Select-Object DeviceID,VolumeName,FileSystem,Size,FreeSpace,DriveType `+
			`| ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(logicalScript)
	if err != nil {
		return nil, fmt.Errorf("failed to get logical disks: %w", err)
	}

	var logicalDisks []win32LogicalDisk
	if err := helpers.ParseJSON(string(output), &logicalDisks); err != nil {
		var single win32LogicalDisk
		if err2 := helpers.ParseJSON(string(output), &single); err2 == nil {
			logicalDisks = []win32LogicalDisk{single}
		} else {
			logicalDisks = []win32LogicalDisk{}
		}
	}

	for _, ld := range logicalDisks {
		used := ld.Size - ld.FreeSpace
		var percent float64
		if ld.Size > 0 {
			percent = float64(used) / float64(ld.Size) * 100.0
		}

		usage := types.DiskUsage{
			Device:     ld.DeviceID,
			MountPoint: ld.DeviceID,
			FSType:     ld.FileSystem,
			Total:      ld.Size,
			Used:       used,
			Free:       ld.FreeSpace,
			Percent:    percent,
		}
		result.Usage = append(result.Usage, usage)
	}

	// Получаем IO счетчики
	ioCounters, err := GetDiskIOCounters()
	if err == nil {
		result.IOCounters = ioCounters
	}

	// Получаем информацию об устройствах
	deviceInfo, err := GetAllDiskDeviceInfo()
	if err == nil {
		result.DeviceInfo = deviceInfo
	}

	return result, nil
}
