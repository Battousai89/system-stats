package stats

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"system-stats/internal/formatter"
)

type NetIOCounters struct {
	Name        string `json:"name"`
	BytesSent   uint64 `json:"bytesSent"`
	BytesRecv   uint64 `json:"bytesRecv"`
	PacketsSent uint64 `json:"packetsSent"`
	PacketsRecv uint64 `json:"packetsRecv"`
	Errin       uint64 `json:"errin"`
	Errout      uint64 `json:"errout"`
	Dropin      uint64 `json:"dropin"`
	Dropout     uint64 `json:"dropout"`
	Fifoin      uint64 `json:"fifoin"`
	Fifoout     uint64 `json:"fifoout"`
}

type NetConnection struct {
	Fd     uint32  `json:"fd"`
	Family uint32  `json:"family"`
	Type   uint32  `json:"type"`
	Laddr  NetAddr `json:"laddr"`
	Raddr  NetAddr `json:"raddr"`
	Status string  `json:"status"`
	Pid    int32   `json:"pid"`
}

type NetAddr struct {
	IP   string `json:"ip"`
	Port uint32 `json:"port"`
}

type NetInterface struct {
	Index        int    `json:"index"`
	Name         string `json:"name"`
	MTU          int    `json:"mtu"`
	Flags        string `json:"flags"`
	HardwareAddr string `json:"hardwareAddr"`
}

func NewNetIOCounters() ([]NetIOCounters, error) {
	switch runtime.GOOS {
	case "linux":
		return parseProcNetDev()
	case "windows":
		return getWindowsNetIO()
	case "darwin", "freebsd":
		return getUnixNetIO()
	default:
		return getGenericNetIO()
	}
}

func parseProcNetDev() ([]NetIOCounters, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []NetIOCounters
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		if name == "lo" {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		bytesRecv, _ := strconv.ParseUint(fields[0], 10, 64)
		packetsRecv, _ := strconv.ParseUint(fields[1], 10, 64)
		errin, _ := strconv.ParseUint(fields[2], 10, 64)
		dropin, _ := strconv.ParseUint(fields[3], 10, 64)
		fifoin, _ := strconv.ParseUint(fields[4], 10, 64)

		bytesSent, _ := strconv.ParseUint(fields[8], 10, 64)
		packetsSent, _ := strconv.ParseUint(fields[9], 10, 64)
		errout, _ := strconv.ParseUint(fields[10], 10, 64)
		dropout, _ := strconv.ParseUint(fields[11], 10, 64)
		fifoout, _ := strconv.ParseUint(fields[12], 10, 64)

		result = append(result, NetIOCounters{
			Name:        name,
			BytesSent:   bytesSent,
			BytesRecv:   bytesRecv,
			PacketsSent: packetsSent,
			PacketsRecv: packetsRecv,
			Errin:       errin,
			Errout:      errout,
			Dropin:      dropin,
			Dropout:     dropout,
			Fifoin:      fifoin,
			Fifoout:     fifoout,
		})
	}

	return result, scanner.Err()
}

func getWindowsNetIO() ([]NetIOCounters, error) {
	output, err := runCommandWithTimeout("wmic", "netuse", "get", "Name,BytesReceived,BytesSent", "/format:csv")
	if err != nil {
		return getGenericNetIO()
	}

	var result []NetIOCounters
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		bytesRecv, _ := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
		bytesSent, _ := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)

		result = append(result, NetIOCounters{
			Name:      strings.TrimSpace(parts[0]),
			BytesSent: bytesSent,
			BytesRecv: bytesRecv,
		})
	}

	return result, nil
}

func getUnixNetIO() ([]NetIOCounters, error) {
	output, err := runCommandWithTimeout("netstat", "-i", "-b")
	if err != nil {
		return getGenericNetIO()
	}

	var result []NetIOCounters
	lines := strings.Split(string(output), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		if fields[0] == "Name" {
			continue
		}

		name := fields[0]
		if name == "lo" || name == "lo0" {
			continue
		}

		bytesRecv, _ := strconv.ParseUint(fields[6], 10, 64)
		bytesSent, _ := strconv.ParseUint(fields[9], 10, 64)
		packetsRecv, _ := strconv.ParseUint(fields[4], 10, 64)
		packetsSent, _ := strconv.ParseUint(fields[7], 10, 64)

		result = append(result, NetIOCounters{
			Name:        name,
			BytesSent:   bytesSent,
			BytesRecv:   bytesRecv,
			PacketsSent: packetsSent,
			PacketsRecv: packetsRecv,
		})
	}

	return result, nil
}

func getGenericNetIO() ([]NetIOCounters, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return []NetIOCounters{}, nil
	}

	var result []NetIOCounters
	for _, iface := range ifaces {
		result = append(result, NetIOCounters{
			Name: iface.Name,
		})
	}

	return result, nil
}

func NewNetConnections(kind string) ([]NetConnection, error) {
	switch runtime.GOOS {
	case "linux":
		return parseNetConnections(kind)
	case "windows":
		return getWindowsConnections(kind)
	case "darwin", "freebsd":
		return getUnixConnections(kind)
	default:
		return []NetConnection{}, nil
	}
}

func parseNetConnections(kind string) ([]NetConnection, error) {
	var result []NetConnection

	if kind == "" || kind == "tcp" || kind == "tcp4" || kind == "tcp6" {
		tcpConns, err := parseTCPConnections(kind)
		if err == nil {
			result = append(result, tcpConns...)
		}
	}

	if kind == "" || kind == "udp" || kind == "udp4" || kind == "udp6" {
		udpConns, err := parseUDPConnections(kind)
		if err == nil {
			result = append(result, udpConns...)
		}
	}

	return result, nil
}

func parseTCPConnections(kind string) ([]NetConnection, error) {
	files := []string{"/proc/net/tcp", "/proc/net/tcp6"}
	if kind == "tcp4" {
		files = []string{"/proc/net/tcp"}
	} else if kind == "tcp6" {
		files = []string{"/proc/net/tcp6"}
	}

	var result []NetConnection
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)

			if len(fields) < 12 || fields[0] == "sl" {
				continue
			}

			laddr, err := parseAddr(fields[1])
			if err != nil {
				continue
			}

			raddr, err := parseAddr(fields[2])
			if err != nil {
				continue
			}

			status := parseTCPStatus(fields[3])

			result = append(result, NetConnection{
				Family: 1, // IPv4 for /proc/net/tcp, IPv6 for tcp6
				Type:   1, // TCP
				Laddr:  laddr,
				Raddr:  raddr,
				Status: status,
			})
		}
		f.Close()
	}

	return result, nil
}

func parseUDPConnections(kind string) ([]NetConnection, error) {
	files := []string{"/proc/net/udp", "/proc/net/udp6"}
	if kind == "udp4" {
		files = []string{"/proc/net/udp"}
	} else if kind == "udp6" {
		files = []string{"/proc/net/udp6"}
	}

	var result []NetConnection
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)

			if len(fields) < 12 || fields[0] == "sl" {
				continue
			}

			laddr, err := parseAddr(fields[1])
			if err != nil {
				continue
			}

			result = append(result, NetConnection{
				Family: 1,
				Type:   2, // UDP
				Laddr:  laddr,
				Status: "NONE",
			})
		}
		f.Close()
	}

	return result, nil
}

func parseAddr(addr string) (NetAddr, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return NetAddr{}, nil
	}

	ipHex := parts[0]
	ip := parseHexIP(ipHex)

	port, err := strconv.ParseUint(parts[1], 16, 32)
	if err != nil {
		return NetAddr{}, err
	}

	return NetAddr{
		IP:   ip,
		Port: uint32(port),
	}, nil
}

func parseHexIP(hex string) string {
	if len(hex) != 8 {
		return "0.0.0.0"
	}

	var ipBytes [4]byte
	for i := 0; i < 4; i++ {
		var b byte
		fmt.Sscanf(hex[i*2:i*2+2], "%02x", &b)
		ipBytes[i] = b
	}

	return strconv.Itoa(int(ipBytes[3])) + "." +
		strconv.Itoa(int(ipBytes[2])) + "." +
		strconv.Itoa(int(ipBytes[1])) + "." +
		strconv.Itoa(int(ipBytes[0]))
}

func parseTCPStatus(status string) string {
	switch status {
	case "01":
		return "ESTABLISHED"
	case "02":
		return "SYN_SENT"
	case "03":
		return "SYN_RECV"
	case "04":
		return "FIN_WAIT1"
	case "05":
		return "FIN_WAIT2"
	case "06":
		return "TIME_WAIT"
	case "07":
		return "CLOSE"
	case "08":
		return "CLOSE_WAIT"
	case "09":
		return "LAST_ACK"
	case "0A":
		return "LISTEN"
	case "0B":
		return "CLOSING"
	default:
		return "UNKNOWN"
	}
}

func getWindowsConnections(kind string) ([]NetConnection, error) {
	output, err := runCommandWithTimeout("netstat", "-ano")
	if err != nil {
		return []NetConnection{}, nil
	}

	var result []NetConnection
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		if fields[0] == "Proto" {
			continue
		}

		proto := strings.ToLower(fields[0])
		if kind != "" && !strings.Contains(kind, proto) {
			continue
		}

		localAddr := parseWindowsAddr(fields[1])
		remoteAddr := parseWindowsAddr(fields[2])
		status := ""
		if len(fields) > 4 {
			status = fields[3]
		}

		result = append(result, NetConnection{
			Type:   getTypeFromProto(proto),
			Laddr:  localAddr,
			Raddr:  remoteAddr,
			Status: status,
		})
	}

	return result, nil
}

func parseWindowsAddr(addr string) NetAddr {
	if strings.Contains(addr, "[") {
		addr = strings.Trim(addr, "[]")
		parts := strings.Split(addr, "]:")
		if len(parts) == 2 {
			port, _ := strconv.ParseUint(parts[1], 10, 32)
			return NetAddr{IP: parts[0], Port: uint32(port)}
		}
	}

	parts := strings.Split(addr, ":")
	if len(parts) == 2 {
		port, _ := strconv.ParseUint(parts[1], 10, 32)
		return NetAddr{IP: parts[0], Port: uint32(port)}
	}

	return NetAddr{IP: addr, Port: 0}
}

func getTypeFromProto(proto string) uint32 {
	if strings.Contains(proto, "tcp") {
		return 1
	}
	return 2
}

func getUnixConnections(kind string) ([]NetConnection, error) {
	output, err := runCommandWithTimeout("netstat", "-an")
	if err != nil {
		return []NetConnection{}, nil
	}

	var result []NetConnection
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		proto := strings.ToLower(fields[0])
		if kind != "" && !strings.Contains(kind, proto) {
			continue
		}

		localAddr := parseUnixAddr(fields[3])
		remoteAddr := parseUnixAddr(fields[4])
		status := ""
		if len(fields) > 5 {
			status = fields[5]
		}

		result = append(result, NetConnection{
			Type:   getTypeFromProto(proto),
			Laddr:  localAddr,
			Raddr:  remoteAddr,
			Status: status,
		})
	}

	return result, nil
}

func parseUnixAddr(addr string) NetAddr {
	parts := strings.Split(addr, ".")
	if len(parts) >= 2 {
		port, _ := strconv.ParseUint(parts[len(parts)-1], 10, 32)
		ip := strings.Join(parts[:len(parts)-1], ".")
		return NetAddr{IP: ip, Port: uint32(port)}
	}
	return NetAddr{IP: addr, Port: 0}
}

func NewNetInterfaces() ([]NetInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	result := make([]NetInterface, 0, len(ifaces))
	for _, i := range ifaces {
		result = append(result, NetInterface{
			Index:        i.Index,
			Name:         i.Name,
			MTU:          i.MTU,
			Flags:        i.Flags.String(),
			HardwareAddr: i.HardwareAddr.String(),
		})
	}
	return result, nil
}

func (c NetIOCounters) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Name", c.Name, "").
		AddField("BytesSent", bytesToHuman(c.BytesSent), "").
		AddField("BytesRecv", bytesToHuman(c.BytesRecv), "").
		AddField("PacketsSent", c.PacketsSent, "").
		AddField("PacketsRecv", c.PacketsRecv, "").
		AddField("Errin", c.Errin, "").
		AddField("Errout", c.Errout, "").
		AddField("Dropin", c.Dropin, "").
		AddField("Dropout", c.Dropout, "").
		Build()
}

func (c NetConnection) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Fd", c.Fd, "").
		AddField("Family", formatFamily(c.Family), "").
		AddField("Type", formatType(c.Type), "").
		AddField("Laddr", c.Laddr.IP+":"+formatPort(c.Laddr.Port), "").
		AddField("Raddr", c.Raddr.IP+":"+formatPort(c.Raddr.Port), "").
		AddField("Status", c.Status, "").
		AddField("Pid", c.Pid, "").
		Build()
}

func (i NetInterface) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Index", i.Index, "").
		AddField("Name", i.Name, "").
		AddField("MTU", i.MTU, "").
		AddField("Flags", i.Flags, "").
		AddField("HardwareAddr", i.HardwareAddr, "").
		Build()
}

func formatPort(port uint32) string {
	if port == 0 {
		return "*"
	}
	return strconv.Itoa(int(port))
}

func formatFamily(family uint32) string {
	switch family {
	case 1:
		return "IPv4"
	case 2:
		return "IPv6"
	default:
		return "unknown"
	}
}

func formatType(typ uint32) string {
	switch typ {
	case 1:
		return "TCP"
	case 2:
		return "UDP"
	default:
		return "unknown"
	}
}

func NetIOCountersToPrint(counters []NetIOCounters) string {
	var sb strings.Builder
	for i, c := range counters {
		sb.WriteString(c.ToPrint())
		if i < len(counters)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func NetConnectionsToPrint(conns []NetConnection) string {
	var sb strings.Builder
	for i, c := range conns {
		sb.WriteString(c.ToPrint())
		if i < len(conns)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func NetInterfacesToPrint(ifaces []NetInterface) string {
	var sb strings.Builder
	for i, iStat := range ifaces {
		sb.WriteString(iStat.ToPrint())
		if i < len(ifaces)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
