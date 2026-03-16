package windows

import (
	"fmt"
	"net"
	"strings"

	"system-stats/internal/constants"
	"system-stats/internal/helpers"
	"system-stats/internal/types"
)

// win32PerfNetInterface структура для сетевых счетчиков
type win32PerfNetInterface struct {
	Name              string  `json:"Name"`
	BytesSentPerSec   float64 `json:"BytesSentPerSec"`
	BytesReceivedPerSec float64 `json:"BytesReceivedPerSec"`
	PacketsSentPerSec float64 `json:"PacketsSentPerSec"`
	PacketsReceivedPerSec float64 `json:"PacketsReceivedPerSec"`
	PacketsOutboundErrors float64 `json:"PacketsOutboundErrors"`
	PacketsReceivedErrors float64 `json:"PacketsReceivedErrors"`
	PacketsOutboundDiscarded float64 `json:"PacketsOutboundDiscarded"`
	PacketsReceivedDiscarded float64 `json:"PacketsReceivedDiscarded"`
}

// win32NetworkAdapter структура для информации об адаптере
type win32NetworkAdapter struct {
	Name            string `json:"Name"`
	NetEnabled      bool   `json:"NetEnabled"`
	NetConnectionID string `json:"NetConnectionID"`
	MACAddress      string `json:"MACAddress"`
	Speed           uint64 `json:"Speed"`
	AdapterTypeID   uint16 `json:"AdapterTypeID"`
}

// win32NetworkAdapterConfiguration структура для конфигурации адаптера
type win32NetworkAdapterConfiguration struct {
	IPAddress       []string `json:"IPAddress"`
	IPSubnet        []string `json:"IPSubnet"`
	DefaultIPGateway []string `json:"DefaultIPGateway"`
	DNSServerSearchOrder []string `json:"DNSServerSearchOrder"`
	DHCPEnabled     bool     `json:"DHCPEnabled"`
	Index           uint32   `json:"Index"`
}

// NewNetIOCounters получает счетчики сетевого I/O
func NewNetIOCounters() ([]types.NetIOCounters, error) {
	script := `
		Get-CimInstance Win32_PerfFormattedData_Tcpip_NetworkInterface | `+
			`Where-Object { $_.Name -ne '_Total' } | `+
			`Select-Object Name,BytesSentPerSec,BytesReceivedPerSec,`+
			`PacketsSentPerSec,PacketsReceivedPerSec,`+
			`PacketsOutboundErrors,PacketsReceivedErrors,`+
			`PacketsOutboundDiscarded,PacketsReceivedDiscarded | `+
			`ConvertTo-Json`

	output, err := helpers.RunPowerShellCommand(script)
	if err != nil {
		return nil, fmt.Errorf("failed to get net IO counters: %w", err)
	}

	var perfNet []win32PerfNetInterface
	if err := helpers.ParseJSON(string(output), &perfNet); err != nil {
		var single win32PerfNetInterface
		if err2 := helpers.ParseJSON(string(output), &single); err2 == nil {
			perfNet = []win32PerfNetInterface{single}
		} else {
			return nil, fmt.Errorf("failed to parse net IO JSON: %w", err)
		}
	}

	result := make([]types.NetIOCounters, 0, len(perfNet))
	for _, p := range perfNet {
		// Пропускаем псевдо-интерфейсы
		if strings.Contains(p.Name, "isatap") || strings.Contains(p.Name, "Teredo") {
			continue
		}

		counter := types.NetIOCounters{
			Name:              p.Name,
			BytesSent:         uint64(p.BytesSentPerSec),
			BytesRecv:         uint64(p.BytesReceivedPerSec),
			PacketsSent:       uint64(p.PacketsSentPerSec),
			PacketsRecv:       uint64(p.PacketsReceivedPerSec),
			Errout:            uint64(p.PacketsOutboundErrors),
			Errin:             uint64(p.PacketsReceivedErrors),
			Dropout:           uint64(p.PacketsOutboundDiscarded),
			Dropin:            uint64(p.PacketsReceivedDiscarded),
			BytesSentPerSec:   uint64(p.BytesSentPerSec),
			BytesRecvPerSec:   uint64(p.BytesReceivedPerSec),
			PacketsSentPerSec: uint64(p.PacketsSentPerSec),
			PacketsRecvPerSec: uint64(p.PacketsReceivedPerSec),
		}

		result = append(result, counter)
	}

	return result, nil
}

// NewNetInterfaces получает информацию о сетевых интерфейсах
func NewNetInterfaces() ([]types.NetInterface, error) {
	// Получаем информацию об адаптерах
	adapterScript := `
		Get-CimInstance Win32_NetworkAdapter | `+
			`Where-Object { $_.NetEnabled -eq $true } | `+
			`Select-Object Name,NetEnabled,NetConnectionID,MACAddress,Speed,AdapterTypeID | `+
			`ConvertTo-Json`

	adapterOutput, err := helpers.RunPowerShellCommand(adapterScript)
	if err != nil {
		return nil, fmt.Errorf("failed to get network adapters: %w", err)
	}

	var adapters []win32NetworkAdapter
	if err := helpers.ParseJSON(string(adapterOutput), &adapters); err != nil {
		var single win32NetworkAdapter
		if err2 := helpers.ParseJSON(string(adapterOutput), &single); err2 == nil {
			adapters = []win32NetworkAdapter{single}
		} else {
			return nil, fmt.Errorf("failed to parse adapters JSON: %w", err)
		}
	}

	// Получаем конфигурацию адаптеров
	configScript := `
		Get-CimInstance Win32_NetworkAdapterConfiguration | `+
			`Where-Object { $_.IPEnabled -eq $true } | `+
			`Select-Object IPAddress,IPSubnet,DefaultIPGateway,DNSServerSearchOrder,DHCPEnabled,Index | `+
			`ConvertTo-Json`

	configOutput, err := helpers.RunPowerShellCommand(configScript)
	if err != nil {
		return nil, fmt.Errorf("failed to get network config: %w", err)
	}

	var configs []win32NetworkAdapterConfiguration
	if err := helpers.ParseJSON(string(configOutput), &configs); err != nil {
		var single win32NetworkAdapterConfiguration
		if err2 := helpers.ParseJSON(string(configOutput), &single); err2 == nil {
			configs = []win32NetworkAdapterConfiguration{single}
		} else {
			configs = []win32NetworkAdapterConfiguration{}
		}
	}

	// Создаем мапу конфигураций по индексу
	configMap := make(map[uint32]*win32NetworkAdapterConfiguration)
	for i := range configs {
		configMap[configs[i].Index] = &configs[i]
	}

	// Получаем системные интерфейсы через Go
	goInterfaces, _ := net.Interfaces()

	result := make([]types.NetInterface, 0)

	// Добавляем адаптеры Windows
	for _, adapter := range adapters {
		iface := types.NetInterface{
			Name:   adapter.NetConnectionID,
			MAC:    adapter.MACAddress,
			Speed:  adapter.Speed,
			Status: "connected",
			DHCP:   false,
		}

		// Пытаемся найти конфигурацию
		// Индексы в Win32_NetworkAdapter и Win32_NetworkAdapterConfiguration могут не совпадать
		for _, cfg := range configs {
			if cfg.IPAddress != nil && len(cfg.IPAddress) > 0 {
				iface.IPAddresses = cfg.IPAddress
				iface.DHCP = cfg.DHCPEnabled
				if cfg.DefaultIPGateway != nil && len(cfg.DefaultIPGateway) > 0 {
					iface.Gateway = cfg.DefaultIPGateway[0]
				}
				if cfg.DNSServerSearchOrder != nil && len(cfg.DNSServerSearchOrder) > 0 {
					iface.DNS = cfg.DNSServerSearchOrder
				}
				break
			}
		}

		// Получаем MTU и флаги из Go интерфейса
		for _, goIface := range goInterfaces {
			if strings.EqualFold(goIface.Name, adapter.Name) ||
				strings.EqualFold(goIface.Name, adapter.NetConnectionID) {
				iface.MTU = goIface.MTU
				iface.Index = goIface.Index
				iface.Flags = flagsToStrings(goIface.Flags)
				break
			}
		}

		result = append(result, iface)
	}

	// Добавляем интерфейсы из Go которые не попали в Win32_NetworkAdapter
	for _, goIface := range goInterfaces {
		found := false
		for _, r := range result {
			if strings.EqualFold(r.Name, goIface.Name) {
				found = true
				break
			}
		}

		if !found {
			addrs, _ := goIface.Addrs()
			ipAddrs := make([]string, 0, len(addrs))
			for _, addr := range addrs {
				ipAddrs = append(ipAddrs, addr.String())
			}

			iface := types.NetInterface{
				Name:        goIface.Name,
				Index:       goIface.Index,
				MTU:         goIface.MTU,
				Flags:       flagsToStrings(goIface.Flags),
				IPAddresses: ipAddrs,
				Status:      "unknown",
			}

			// Проверяем, активен ли интерфейс
			if goIface.Flags&net.FlagUp != 0 {
				iface.Status = "up"
			} else {
				iface.Status = "down"
			}

			result = append(result, iface)
		}
	}

	return result, nil
}

// NewNetProtocolCounters получает счетчики протоколов
func NewNetProtocolCounters() ([]types.NetProtocolCounters, error) {
	result := make([]types.NetProtocolCounters, 0, 3)

	// TCP счетчики
	tcpScript := `
		Get-CimInstance ` + constants.Win32PerfFormattedDataTcpipTCPv4 + ` | `+
			`Select-Object ConnectionsEstablished,SegmentsSent,SegmentsReceived | `+
			`ConvertTo-Json`
	tcpOutput, tcpErr := helpers.RunPowerShellCommand(tcpScript)
	if tcpErr == nil {
		var tcp struct {
			ConnectionsEstablished uint32 `json:"ConnectionsEstablished"`
			SegmentsSent           uint32 `json:"SegmentsSent"`
			SegmentsReceived       uint32 `json:"SegmentsReceived"`
		}
		if err := helpers.ParseJSON(string(tcpOutput), &tcp); err == nil {
			result = append(result, types.NetProtocolCounters{
				Protocol:     "TCPv4",
				Connections:  uint64(tcp.ConnectionsEstablished),
				SegmentsSent: uint64(tcp.SegmentsSent),
				SegmentsRecv: uint64(tcp.SegmentsReceived),
			})
		}
	}

	// UDP счетчики
	udpScript := `
		Get-CimInstance ` + constants.Win32PerfFormattedDataTcpipUDPv4 + ` | `+
			`Select-Object DatagramsSent,DatagramsReceived | `+
			`ConvertTo-Json`
	udpOutput, udpErr := helpers.RunPowerShellCommand(udpScript)
	if udpErr == nil {
		var udp struct {
			DatagramsSent     uint32 `json:"DatagramsSent"`
			DatagramsReceived uint32 `json:"DatagramsReceived"`
		}
		if err := helpers.ParseJSON(string(udpOutput), &udp); err == nil {
			result = append(result, types.NetProtocolCounters{
				Protocol:    "UDPv4",
				PacketsSent: uint64(udp.DatagramsSent),
				PacketsRecv: uint64(udp.DatagramsReceived),
			})
		}
	}

	// IP счетчики
	ipScript := `
		Get-CimInstance ` + constants.Win32PerfFormattedDataTcpipIPv4 + ` | `+
			`Select-Object PacketsSent,PacketsReceived,PacketsOutboundErrors,`+
			`PacketsReceivedErrors,PacketsOutboundDiscarded,PacketsReceivedDiscarded | `+
			`ConvertTo-Json`
	ipOutput, ipErr := helpers.RunPowerShellCommand(ipScript)
	if ipErr == nil {
		var ip struct {
			PacketsSent            uint32 `json:"PacketsSent"`
			PacketsReceived        uint32 `json:"PacketsReceived"`
			PacketsOutboundErrors  uint32 `json:"PacketsOutboundErrors"`
			PacketsReceivedErrors  uint32 `json:"PacketsReceivedErrors"`
			PacketsOutboundDiscarded uint32 `json:"PacketsOutboundDiscarded"`
			PacketsReceivedDiscarded uint32 `json:"PacketsReceivedDiscarded"`
		}
		if err := helpers.ParseJSON(string(ipOutput), &ip); err == nil {
			result = append(result, types.NetProtocolCounters{
				Protocol:    "IPv4",
				PacketsSent: uint64(ip.PacketsSent),
				PacketsRecv: uint64(ip.PacketsReceived),
				Errors:      uint64(ip.PacketsOutboundErrors + ip.PacketsReceivedErrors),
				Dropped:     uint64(ip.PacketsOutboundDiscarded + ip.PacketsReceivedDiscarded),
			})
		}
	}

	return result, nil
}

// flagsToStrings конвертирует флаги интерфейса в строки
func flagsToStrings(flags net.Flags) []string {
	result := make([]string, 0)

	if flags&net.FlagUp != 0 {
		result = append(result, "up")
	}
	if flags&net.FlagBroadcast != 0 {
		result = append(result, "broadcast")
	}
	if flags&net.FlagLoopback != 0 {
		result = append(result, "loopback")
	}
	if flags&net.FlagPointToPoint != 0 {
		result = append(result, "pointtopoint")
	}
	if flags&net.FlagMulticast != 0 {
		result = append(result, "multicast")
	}

	return result
}
