package types

import (
	"fmt"
	"strings"

	"system-stats/internal/helpers"
)

// NetIOCounters счетчики сетевого интерфейса
type NetIOCounters struct {
	Name        string `json:"name"`         // Имя интерфейса
	BytesSent   uint64 `json:"bytes_sent"`   // Отправлено байт
	BytesRecv   uint64 `json:"bytes_recv"`   // Получено байт
	PacketsSent uint64 `json:"packets_sent"` // Отправлено пакетов
	PacketsRecv uint64 `json:"packets_recv"` // Получено пакетов
	Errin       uint64 `json:"errin"`        // Ошибок при получении
	Errout      uint64 `json:"errout"`       // Ошибок при отправке
	Dropin      uint64 `json:"dropin"`       // Потеряно при получении
	Dropout     uint64 `json:"dropout"`      // Потеряно при отправке
	Fifoin      uint64 `json:"fifoin"`       // FIFO ошибок при получении
	Fifoout     uint64 `json:"fifoout"`      // FIFO ошибок при отправке
	BytesSentPerSec  uint64 `json:"bytes_sent_per_sec"`  // Отправлено байт/сек
	BytesRecvPerSec  uint64 `json:"bytes_recv_per_sec"`  // Получено байт/сек
	PacketsSentPerSec uint64 `json:"packets_sent_per_sec"` // Отправлено пакетов/сек
	PacketsRecvPerSec uint64 `json:"packets_recv_per_sec"` // Получено пакетов/сек
}

// ToPrint форматирует NetIOCounters для вывода
func (n *NetIOCounters) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Name", n.Name, "")
	b.AddFieldWithFormatter("Bytes Sent", n.BytesSent, "", formatBytes)
	b.AddFieldWithFormatter("Bytes Recv", n.BytesRecv, "", formatBytes)
	b.AddField("Packets Sent", n.PacketsSent, "")
	b.AddField("Packets Recv", n.PacketsRecv, "")
	
	if n.BytesSentPerSec > 0 || n.BytesRecvPerSec > 0 {
		b.AddFieldWithFormatter("Bytes Sent/sec", n.BytesSentPerSec, "", formatBytes)
		b.AddFieldWithFormatter("Bytes Recv/sec", n.BytesRecvPerSec, "", formatBytes)
	}
	if n.PacketsSentPerSec > 0 || n.PacketsRecvPerSec > 0 {
		b.AddField("Packets Sent/sec", n.PacketsSentPerSec, "")
		b.AddField("Packets Recv/sec", n.PacketsRecvPerSec, "")
	}
	
	if n.Errin > 0 || n.Errout > 0 {
		b.AddField("Errors In", n.Errin, "")
		b.AddField("Errors Out", n.Errout, "")
	}
	if n.Dropin > 0 || n.Dropout > 0 {
		b.AddField("Dropped In", n.Dropin, "")
		b.AddField("Dropped Out", n.Dropout, "")
	}

	return b.Build()
}

// NetIOCountersToPrint форматирует список NetIOCounters
func NetIOCountersToPrint(counters []NetIOCounters) string {
	var sb strings.Builder
	for i, c := range counters {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("  Interface %d:\n", i+1))
		sb.WriteString(c.ToPrint())
	}
	return sb.String()
}

// NetInterface информация о сетевом интерфейсе
type NetInterface struct {
	Name        string   `json:"name"`        // Имя интерфейса
	Index       int      `json:"index"`       // Индекс интерфейса
	MTU         int      `json:"mtu"`         // MTU
	MAC         string   `json:"mac"`         // MAC адрес
	Flags       []string `json:"flags"`       // Флаги (up, broadcast, multicast, etc.)
	IPAddresses []string `json:"ip_addresses"` // IP адреса
	Gateway     string   `json:"gateway"`     // Шлюз по умолчанию
	DNS         []string `json:"dns"`         // DNS серверы
	DHCP        bool     `json:"dhcp"`        // Использует DHCP
	Status      string   `json:"status"`      // Статус (connected, disconnected)
	Speed       uint64   `json:"speed"`       // Скорость соединения (бит/сек)
}

// ToPrint форматирует NetInterface для вывода
func (n *NetInterface) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Name", n.Name, "")
	b.AddField("Index", n.Index, "")
	b.AddField("MTU", n.MTU, "")
	
	if n.MAC != "" {
		b.AddField("MAC", n.MAC, "")
	}
	
	if len(n.IPAddresses) > 0 {
		b.AddField("IP Addresses", strings.Join(n.IPAddresses, ", "), "")
	}
	
	if n.Gateway != "" {
		b.AddField("Gateway", n.Gateway, "")
	}
	
	if len(n.DNS) > 0 {
		b.AddField("DNS", strings.Join(n.DNS, ", "), "")
	}
	
	if n.DHCP {
		b.AddField("DHCP", "Enabled", "")
	}
	
	b.AddField("Status", n.Status, "")
	
	if n.Speed > 0 {
		speedStr := formatSpeed(n.Speed)
		b.AddField("Speed", speedStr, "")
	}

	return b.Build()
}

// NetInterfacesToPrint форматирует список NetInterface
func NetInterfacesToPrint(ifaces []NetInterface) string {
	var sb strings.Builder
	for i, iface := range ifaces {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("  Interface %d:\n", i+1))
		sb.WriteString(iface.ToPrint())
	}
	return sb.String()
}

// NetProtocolCounters счетчики протоколов
type NetProtocolCounters struct {
	Protocol    string `json:"protocol"`     // Протокол (TCP, UDP, IP, ICMP)
	PacketsSent uint64 `json:"packets_sent"` // Отправлено пакетов
	PacketsRecv uint64 `json:"packets_recv"` // Получено пакетов
	Errors      uint64 `json:"errors"`       // Ошибки
	Dropped     uint64 `json:"dropped"`      // Потеряно
	SegmentsSent uint64 `json:"segments_sent"` // Отправлено сегментов (TCP)
	SegmentsRecv uint64 `json:"segments_recv"` // Получено сегментов (TCP)
	Connections uint64 `json:"connections"`  // Активные соединения (TCP)
}

// ToPrint форматирует NetProtocolCounters для вывода
func (n *NetProtocolCounters) ToPrint() string {
	b := helpers.NewBuilder()

	b.AddField("Protocol", n.Protocol, "")
	b.AddField("Packets Sent", n.PacketsSent, "")
	b.AddField("Packets Recv", n.PacketsRecv, "")
	
	if n.SegmentsSent > 0 || n.SegmentsRecv > 0 {
		b.AddField("Segments Sent", n.SegmentsSent, "")
		b.AddField("Segments Recv", n.SegmentsRecv, "")
	}
	if n.Connections > 0 {
		b.AddField("Active Connections", n.Connections, "")
	}
	if n.Errors > 0 {
		b.AddField("Errors", n.Errors, "")
	}
	if n.Dropped > 0 {
		b.AddField("Dropped", n.Dropped, "")
	}

	return b.Build()
}

// NetProtocolCountersToPrint форматирует список NetProtocolCounters
func NetProtocolCountersToPrint(counters []NetProtocolCounters) string {
	var sb strings.Builder
	for i, c := range counters {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("  %s:\n", c.Protocol))
		sb.WriteString(c.ToPrint())
	}
	return sb.String()
}

// formatSpeed форматирует скорость в человекочитаемый формат
func formatSpeed(bps uint64) string {
	const (
		Kbps = 1000
		Mbps = Kbps * 1000
		Gbps = Mbps * 1000
	)

	switch {
	case bps >= Gbps:
		return fmt.Sprintf("%.2f Gbps", float64(bps)/Gbps)
	case bps >= Mbps:
		return fmt.Sprintf("%.2f Mbps", float64(bps)/Mbps)
	case bps >= Kbps:
		return fmt.Sprintf("%.2f Kbps", float64(bps)/Kbps)
	default:
		return fmt.Sprintf("%d bps", bps)
	}
}
