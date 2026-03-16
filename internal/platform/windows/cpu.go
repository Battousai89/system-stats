package windows

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"system-stats/internal/constants"
	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// win32Processor структура для парсинга вывода Win32_Processor
type win32Processor struct {
	Name                    string  `json:"Name"`
	Manufacturer            string  `json:"Manufacturer"`
	Family                  any     `json:"Family"`
	Model                   any     `json:"Model"`
	Stepping                string  `json:"Stepping"`
	Architecture            any     `json:"Architecture"`
	SocketDesignation       string  `json:"SocketDesignation"`
	L2CacheSize             any     `json:"L2CacheSize"`
	L3CacheSize             any     `json:"L3CacheSize"`
	NumberOfCores           uint32  `json:"NumberOfCores"`
	NumberOfLogicalProcessors uint32 `json:"NumberOfLogicalProcessors"`
	CurrentClockSpeed       uint64  `json:"CurrentClockSpeed"`
	MaxClockSpeed           uint64  `json:"MaxClockSpeed"`
	CurrentVoltage          any     `json:"CurrentVoltage"`
	LoadPercentage          any     `json:"LoadPercentage"`
	ProcessorType           any     `json:"ProcessorType"`
	Status                  string  `json:"Status"`
	ConfigManagerErrorCode  any     `json:"ConfigManagerErrorCode"`
	Caption                 string  `json:"Caption"`
	DeviceID                string  `json:"DeviceID"`
	Description             string  `json:"Description"`
}

// win32PerformanceCounter структура для счетчиков производительности
type win32PerformanceCounter struct {
	Name                string  `json:"Name"`
	PercentProcessorTime float64 `json:"PercentProcessorTime"`
	PercentUserTime     float64 `json:"PercentUserTime"`
	PercentPrivilegedTime float64 `json:"PercentPrivilegedTime"`
	PercentIdleTime     float64 `json:"PercentIdleTime"`
	Timestamp_Sys       uint64  `json:"Timestamp_Sys"`
}

// win32OSTime структура для времен ОС
type win32OSTime struct {
	TotalProcessorTime      any `json:"TotalProcessorTime"`
	UserProcessorTime       any `json:"UserProcessorTime"`
	PrivilegedProcessorTime any `json:"PrivilegedProcessorTime"`
	InterruptProcessorTime  any `json:"InterruptProcessorTime"`
	DPCProcessorTime        any `json:"DPCProcessorTime"`
	IdleProcessorTime       any `json:"IdleProcessorTime"`
}

// NewCPUInfo получает информацию о процессоре
func NewCPUInfo() ([]types.CPUInfo, error) {
	// Объединенный запрос для CPU info и температуры
	script := `
		$cpu = Get-CimInstance ` + constants.Win32Processor + `
		$temp = Get-CimInstance -Namespace ` + constants.RootWMI + ` ` + constants.MSAcpiThermalZoneTemperature + ` 2>$null | Select-Object -First 1 CurrentTemperature
		[PSCustomObject]@{
			Processors = $cpu | Select-Object Name,Manufacturer,Family,Model,Stepping,Architecture,` +
		`SocketDesignation,L2CacheSize,L3CacheSize,` +
		`NumberOfCores,NumberOfLogicalProcessors,` +
		`CurrentClockSpeed,MaxClockSpeed,CurrentVoltage,` +
		`LoadPercentage,ProcessorType,Status,` +
		`ConfigManagerErrorCode,Caption,DeviceID,Description
			Temperature = if ($temp) { $temp.CurrentTemperature } else { $null }
		} | ConvertTo-Json -Depth 3`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	var result struct {
		Processors  any     `json:"Processors"`
		Temperature *uint32 `json:"Temperature"`
	}

	if err := helpers.ParseJSON(string(output), &result); err != nil {
		return nil, fmt.Errorf("failed to parse CPU info JSON: %w", err)
	}

	// Обрабатываем случай с одним процессором (возвращается как объект, а не массив)
	var processors []win32Processor
	switch v := result.Processors.(type) {
	case []interface{}:
		// Массив процессоров
		jsonData, _ := json.Marshal(v)
		if err := helpers.ParseJSON(string(jsonData), &processors); err != nil {
			return nil, fmt.Errorf("failed to parse processors array: %w", err)
		}
	case interface{}:
		// Один процессор
		var single win32Processor
		jsonData, _ := json.Marshal(v)
		if err := helpers.ParseJSON(string(jsonData), &single); err != nil {
			return nil, fmt.Errorf("failed to parse single processor: %w", err)
		}
		processors = []win32Processor{single}
	}

	// Конвертируем температуру из Кельви в Цельсии
	var temperature uint32
	if result.Temperature != nil && *result.Temperature > 0 {
		temperature = uint32(float64(*result.Temperature)/10.0 - 273.15)
	}

	cpuResult := make([]types.CPUInfo, 0, len(processors))
	for _, p := range processors {
		cpu := types.CPUInfo{
			Name:                    p.Name,
			Manufacturer:            p.Manufacturer,
			Family:                  interfaceToString(p.Family),
			Model:                   interfaceToString(p.Model),
			Stepping:                p.Stepping,
			Architecture:            architectureToString(p.Architecture),
			Socket:                  p.SocketDesignation,
			L2CacheSize:             interfaceToUint32(p.L2CacheSize),
			L3CacheSize:             interfaceToUint32(p.L3CacheSize),
			Cores:                   p.NumberOfCores,
			LogicalProcessors:       p.NumberOfLogicalProcessors,
			CurrentClockSpeed:       p.CurrentClockSpeed,
			MaxClockSpeed:           p.MaxClockSpeed,
			Voltage:                 voltageToString(p.CurrentVoltage),
			LoadPercentage:          interfaceToUint8(p.LoadPercentage),
			ProcessorType:           processorTypeToString(p.ProcessorType),
			Status:                  statusToString(p.Status, p.ConfigManagerErrorCode),
			Enabled:                 configManagerErrorToBool(p.ConfigManagerErrorCode),
			Caption:                 p.Caption,
			DeviceID:                p.DeviceID,
			NumberOfCores:           p.NumberOfCores,
			NumberOfLogicalProcessors: p.NumberOfLogicalProcessors,
		}

		if temperature > 0 {
			cpu.Temperature = temperature
		}

		cpuResult = append(cpuResult, cpu)
	}

	if len(cpuResult) == 0 {
		return nil, fmt.Errorf("no CPU information found")
	}

	return cpuResult, nil
}

// NewCPUTimes получает времена процессора
func NewCPUTimes() ([]types.CPUTimes, error) {
	script := `
		Get-CimInstance ` + constants.Win32PerfFormattedDataPerfOSProcessor + ` | `+
			`Select-Object Name,PercentProcessorTime,PercentUserTime,`+
			`PercentPrivilegedTime,PercentInterruptTime,PercentDPCTime `+
			`| ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU times: %w", err)
	}

	var rawData []struct {
		Name                  string `json:"Name"`
		PercentProcessorTime  any    `json:"PercentProcessorTime"`
		PercentUserTime       any    `json:"PercentUserTime"`
		PercentPrivilegedTime any    `json:"PercentPrivilegedTime"`
		PercentInterruptTime  any    `json:"PercentInterruptTime"`
		PercentDPCTime        any    `json:"PercentDPCTime"`
	}

	if err := helpers.ParseJSON(string(output), &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse CPU times JSON: %w", err)
	}

	result := make([]types.CPUTimes, 0, len(rawData))
	for _, r := range rawData {
		// Для Win32_PerfFormattedData значения уже в процентах
		percentProc := interfaceToFloat64(r.PercentProcessorTime)
		percentUser := interfaceToFloat64(r.PercentUserTime)
		percentPriv := interfaceToFloat64(r.PercentPrivilegedTime)
		percentInterrupt := interfaceToFloat64(r.PercentInterruptTime)
		percentDPC := interfaceToFloat64(r.PercentDPCTime)

		// Конвертируем проценты во времена (предполагаем 1 секунду измерения)
		userSec := percentUser / 100.0
		systemSec := percentPriv / 100.0
		interruptSec := percentInterrupt / 100.0
		dpcSec := percentDPC / 100.0
		idleSec := 1.0 - (percentProc / 100.0)
		if idleSec < 0 {
			idleSec = 0
		}

		cpuTime := types.CPUTimes{
			CPU:       r.Name,
			User:      userSec,
			System:    systemSec,
			Idle:      idleSec,
			Interrupt: interruptSec,
			DPC:       dpcSec,
			Total:     userSec + systemSec + idleSec + interruptSec + dpcSec,
			Usage:     percentProc,
		}

		result = append(result, cpuTime)
	}

	return result, nil
}

// interruptDPCTimes времена прерываний и DPC
type interruptDPCTimes struct {
	Interrupt float64
	DPC       float64
}

// getInterruptDPCTimes получает времена прерываний и DPC
func getInterruptDPCTimes() (interruptDPCTimes, error) {
	script := `
		Get-CimInstance ` + constants.Win32PerfRawDataPerfOSSystem + ` `+
			`| Select-Object PercentTimeProcessingInterrupts,`+
			`PercentTimeReceivingInterrupts,PercentTimeRunningDPCs `+
			`| ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return interruptDPCTimes{}, err
	}

	var data struct {
		PercentTimeProcessingInterrupts any `json:"PercentTimeProcessingInterrupts"`
		PercentTimeReceivingInterrupts  any `json:"PercentTimeReceivingInterrupts"`
		PercentTimeRunningDPCs          any `json:"PercentTimeRunningDPCs"`
	}

	if err := helpers.ParseJSON(string(output), &data); err != nil {
		return interruptDPCTimes{}, err
	}

	interrupt := interfaceToFloat64(data.PercentTimeProcessingInterrupts) +
		interfaceToFloat64(data.PercentTimeReceivingInterrupts)
	dpc := interfaceToFloat64(data.PercentTimeRunningDPCs)

	return interruptDPCTimes{
		Interrupt: interrupt / 100.0,
		DPC:       dpc / 100.0,
	}, nil
}

// NewCPUPercent получает процент использования процессора
func NewCPUPercent() ([]types.CPUPercent, error) {
	// Используем Win32_PerfFormattedData для готовых процентов
	script := `
		Get-CimInstance ` + constants.Win32PerfFormattedDataPerfOSProcessor + ` | `+
			`Select-Object Name,PercentProcessorTime,PercentUserTime,`+
			`PercentPrivilegedTime `+
			`| ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU percent: %w", err)
	}

	var data []rawCPUData
	if err := helpers.ParseJSON(string(output), &data); err != nil {
		var single rawCPUData
		if err2 := helpers.ParseJSON(string(output), &single); err2 == nil {
			data = []rawCPUData{single}
		} else {
			return nil, fmt.Errorf("failed to parse CPU percent JSON: %w", err)
		}
	}

	result := make([]types.CPUPercent, 0, len(data))
	for _, d := range data {
		if d.Name == "" || d.Name == "_Total" {
			continue
		}

		totalPercent := d.PercentProcessorTime
		userPercent := d.PercentUserTime
		systemPercent := d.PercentPrivilegedTime
		idlePercent := 100.0 - totalPercent

		if idlePercent < 0 {
			idlePercent = 0
		}

		result = append(result, types.CPUPercent{
			CPU:           d.Name,
			Percent:       totalPercent,
			UserPercent:   userPercent,
			SystemPercent: systemPercent,
			IdlePercent:   idlePercent,
		})
	}

	return result, nil
}

// rawCPUData сырые данные о загрузке CPU (используется в NewCPUPercent)
type rawCPUData struct {
	Name                string  `json:"Name"`
	PercentProcessorTime float64 `json:"PercentProcessorTime"`
	PercentUserTime     float64 `json:"PercentUserTime"`
	PercentPrivilegedTime float64 `json:"PercentPrivilegedTime"`
}

// Вспомогательные функции конвертации

func interfaceToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func interfaceToUint32(v any) uint32 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return uint32(val)
	case int:
		return uint32(val)
	case uint:
		return uint32(val)
	case uint32:
		return val
	case int32:
		return uint32(val)
	case string:
		if n, err := strconv.ParseUint(val, 10, 32); err == nil {
			return uint32(n)
		}
	default:
		return 0
	}
	return 0
}

func interfaceToUint8(v any) uint8 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return uint8(val)
	case int:
		return uint8(val)
	case uint:
		return uint8(val)
	case uint32:
		return uint8(val)
	case uint8:
		return val
	case string:
		if n, err := strconv.ParseUint(val, 10, 8); err == nil {
			return uint8(n)
		}
	default:
		return 0
	}
	return 0
}

func interfaceToFloat64(v any) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case uint:
		return float64(val)
	case uint32:
		return float64(val)
	case string:
		if n, err := strconv.ParseFloat(val, 64); err == nil {
			return n
		}
	default:
		return 0
	}
	return 0
}

func architectureToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		switch uint16(val) {
		case 0:
			return "x86"
		case 1:
			return "MIPS"
		case 2:
			return "Alpha"
		case 3:
			return "PowerPC"
		case 5:
			return "ARM"
		case 6:
			return "ia64"
		case 9:
			return "x64"
		case 12:
			return "ARM64"
		default:
			return fmt.Sprintf("Unknown (%d)", uint16(val))
		}
	default:
		return fmt.Sprintf("%v", val)
	}
}

func voltageToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case float64:
		// Напряжение в десятых вольта
		return fmt.Sprintf("%.1f", val/10.0)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

func processorTypeToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		switch uint16(val) {
		case 1:
			return "Other"
		case 2:
			return "Unknown"
		case 3:
			return "Central Processor"
		case 4:
			return "Math Processor"
		case 5:
			return "DSP Processor"
		case 6:
			return "Video Processor"
		default:
			return fmt.Sprintf("Unknown (%d)", uint16(val))
		}
	default:
		return fmt.Sprintf("%v", val)
	}
}

func statusToString(status string, errorCode any) string {
	if status == "OK" {
		return "OK"
	}
	if errorCode != nil {
		switch val := errorCode.(type) {
		case float64:
			if val == 0 {
				return "OK"
			}
			return fmt.Sprintf("Error Code: %d", uint32(val))
		}
	}
	if status != "" {
		return status
	}
	return "Unknown"
}

func configManagerErrorToBool(errorCode any) bool {
	if errorCode == nil {
		return true
	}
	switch val := errorCode.(type) {
	case float64:
		return val == 0
	case int:
		return val == 0
	case uint32:
		return val == 0
	default:
		return true
	}
}

// GetCPUInfoFromCache возвращает кэшированную информацию о CPU (для быстрого доступа)
var cachedCPUInfo []types.CPUInfo
var cacheTime time.Time

// GetCachedCPUInfo получает информацию о CPU с кэшированием
func GetCachedCPUInfo() ([]types.CPUInfo, error) {
	// Кэшируем на 5 секунд
	if cachedCPUInfo != nil && time.Since(cacheTime) < 5*time.Second {
		return cachedCPUInfo, nil
	}

	info, err := NewCPUInfo()
	if err != nil {
		return nil, err
	}

	cachedCPUInfo = info
	cacheTime = time.Now()
	return info, nil
}

// GetCPUCoreCount возвращает количество ядер процессора
func GetCPUCoreCount() (uint32, error) {
	info, err := GetCachedCPUInfo()
	if err != nil {
		return 0, err
	}

	var totalCores uint32
	for _, cpu := range info {
		totalCores += cpu.Cores
	}
	return totalCores, nil
}

// GetCPUThreadCount возвращает количество потоков процессора
func GetCPUThreadCount() (uint32, error) {
	info, err := GetCachedCPUInfo()
	if err != nil {
		return 0, err
	}

	var totalThreads uint32
	for _, cpu := range info {
		totalThreads += cpu.LogicalProcessors
	}
	return totalThreads, nil
}

// GetCPUModelName возвращает модель процессора
func GetCPUModelName() (string, error) {
	info, err := GetCachedCPUInfo()
	if err != nil {
		return "", err
	}

	if len(info) > 0 {
		return info[0].Name, nil
	}
	return "", fmt.Errorf("no CPU info available")
}

// GetAllCPUStats возвращает всю информацию о CPU в одной структуре
type AllCPUStats struct {
	Info       []types.CPUInfo   `json:"info"`
	Times      []types.CPUTimes  `json:"times"`
	Percent    []types.CPUPercent `json:"percent"`
	CoreCount  uint32            `json:"core_count"`
	ThreadCount uint32           `json:"thread_count"`
}

// GetAllCPUStats собирает всю информацию о CPU
func GetAllCPUStats() (*AllCPUStats, error) {
	result := &AllCPUStats{}

	info, err := NewCPUInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}
	result.Info = info

	times, err := NewCPUTimes()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU times: %w", err)
	}
	result.Times = times

	percent, err := NewCPUPercent()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU percent: %w", err)
	}
	result.Percent = percent

	result.CoreCount = func() uint32 {
		var total uint32
		for _, cpu := range info {
			total += cpu.Cores
		}
		return total
	}()

	result.ThreadCount = func() uint32 {
		var total uint32
		for _, cpu := range info {
			total += cpu.LogicalProcessors
		}
		return total
	}()

	return result, nil
}
