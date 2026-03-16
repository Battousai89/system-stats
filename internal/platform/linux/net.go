//go:build linux
// +build linux

package linux

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"system-stats/internal/types"
)

// NewNetIOCounters gets network I/O counters on Linux
func NewNetIOCounters() ([]types.NetIOCounters, error) {
	content, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/net/dev: %w", err)
	}

	return parseNetDev(content)
}

// NewNetInterfaces gets network interface information on Linux
func NewNetInterfaces() ([]types.NetInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	result := make([]types.NetInterface, 0, len(interfaces))

	for _, iface := range interfaces {
		netIface := types.NetInterface{
			Name:  iface.Name,
			Index: iface.Index,
			MTU:   iface.MTU,
			MAC:   iface.HardwareAddr.String(),
			Flags: flagsToStrings(iface.Flags),
		}

		// Get IP addresses
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				netIface.IPAddresses = append(netIface.IPAddresses, addr.String())
			}
		}

		// Get status
		if iface.Flags&net.FlagUp != 0 {
			netIface.Status = "up"
		} else {
			netIface.Status = "down"
		}

		// Get speed if available
		netIface.Speed = getInterfaceSpeed(iface.Name)

		// Get gateway and DNS from system configuration
		netIface.Gateway = getDefaultGateway()
		netIface.DNS = getDNServers()

		result = append(result, netIface)
	}

	return result, nil
}

// NewNetProtocolCounters gets network protocol counters on Linux
func NewNetProtocolCounters() ([]types.NetProtocolCounters, error) {
	var result []types.NetProtocolCounters

	// Get TCP stats
	if tcp, err := getTCPStats(); err == nil {
		result = append(result, tcp)
	}

	// Get UDP stats
	if udp, err := getUDPStats(); err == nil {
		result = append(result, udp)
	}

	// Get IP stats
	if ip, err := getIPStats(); err == nil {
		result = append(result, ip)
	}

	return result, nil
}

// parseNetDev parses /proc/net/dev
func parseNetDev(content []byte) ([]types.NetIOCounters, error) {
	var result []types.NetIOCounters

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	
	// Skip header lines (first 2 lines)
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		ifaceName := strings.TrimSpace(parts[0])
		stats := strings.Fields(parts[1])

		if len(stats) < 16 {
			continue
		}

		// Skip loopback
		if ifaceName == "lo" {
			continue
		}

		bytesRecv, _ := strconv.ParseUint(stats[0], 10, 64)
		packetsRecv, _ := strconv.ParseUint(stats[1], 10, 64)
		errIn, _ := strconv.ParseUint(stats[2], 10, 64)
		dropIn, _ := strconv.ParseUint(stats[3], 10, 64)
		
		bytesSent, _ := strconv.ParseUint(stats[8], 10, 64)
		packetsSent, _ := strconv.ParseUint(stats[9], 10, 64)
		errOut, _ := strconv.ParseUint(stats[10], 10, 64)
		dropOut, _ := strconv.ParseUint(stats[11], 10, 64)

		counter := types.NetIOCounters{
			Name:        ifaceName,
			BytesRecv:   bytesRecv,
			BytesSent:   bytesSent,
			PacketsRecv: packetsRecv,
			PacketsSent: packetsSent,
			Errin:       errIn,
			Errout:      errOut,
			Dropin:      dropIn,
			Dropout:     dropOut,
		}

		result = append(result, counter)
	}

	return result, nil
}

// getTCPStats gets TCP protocol statistics
func getTCPStats() (types.NetProtocolCounters, error) {
	content, err := os.ReadFile("/proc/net/snmp")
	if err != nil {
		return types.NetProtocolCounters{}, err
	}

	var tcpLine string
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Tcp:") {
			if tcpLine == "" {
				tcpLine = line
			} else {
				// Second line - this is the values
				return parseTCPStats(tcpLine, line)
			}
		}
	}

	return types.NetProtocolCounters{}, fmt.Errorf("TCP stats not found")
}

// parseTCPStats parses TCP statistics
func parseTCPStats(headerLine, valueLine string) (types.NetProtocolCounters, error) {
	headers := strings.Fields(headerLine)
	values := strings.Fields(valueLine)

	if len(headers) != len(values) {
		return types.NetProtocolCounters{}, fmt.Errorf("header/value mismatch")
	}

	tcp := types.NetProtocolCounters{
		Protocol: "TCPv4",
	}

	for i, header := range headers {
		value, _ := strconv.ParseUint(values[i], 10, 64)
		
		switch header {
		case "ActiveOpens":
			// Active connections opened
		case "PassiveOpens":
			// Passive connections opened
		case "CurrEstab":
			tcp.Connections = value
		case "InSegs":
			tcp.SegmentsRecv = value
		case "OutSegs":
			tcp.SegmentsSent = value
		case "RetransSegs":
			// Retransmitted segments
		case "InErrs":
			tcp.Errors = value
		}
	}

	return tcp, nil
}

// getUDPStats gets UDP protocol statistics
func getUDPStats() (types.NetProtocolCounters, error) {
	content, err := os.ReadFile("/proc/net/snmp")
	if err != nil {
		return types.NetProtocolCounters{}, err
	}

	var udpLine string
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Udp:") {
			if udpLine == "" {
				udpLine = line
			} else {
				// Second line - this is the values
				return parseUDPStats(udpLine, line)
			}
		}
	}

	return types.NetProtocolCounters{}, fmt.Errorf("UDP stats not found")
}

// parseUDPStats parses UDP statistics
func parseUDPStats(headerLine, valueLine string) (types.NetProtocolCounters, error) {
	headers := strings.Fields(headerLine)
	values := strings.Fields(valueLine)

	udp := types.NetProtocolCounters{
		Protocol: "UDPv4",
	}

	for i, header := range headers {
		if i >= len(values) {
			continue
		}
		value, _ := strconv.ParseUint(values[i], 10, 64)
		
		switch header {
		case "InDatagrams":
			udp.PacketsRecv = value
		case "OutDatagrams":
			udp.PacketsSent = value
		case "InErrors":
			udp.Errors = value
		}
	}

	return udp, nil
}

// getIPStats gets IP protocol statistics
func getIPStats() (types.NetProtocolCounters, error) {
	content, err := os.ReadFile("/proc/net/snmp")
	if err != nil {
		return types.NetProtocolCounters{}, err
	}

	var ipLine string
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Ip:") {
			if ipLine == "" {
				ipLine = line
			} else {
				// Second line - this is the values
				return parseIPStats(ipLine, line)
			}
		}
	}

	return types.NetProtocolCounters{}, fmt.Errorf("IP stats not found")
}

// parseIPStats parses IP statistics
func parseIPStats(headerLine, valueLine string) (types.NetProtocolCounters, error) {
	headers := strings.Fields(headerLine)
	values := strings.Fields(valueLine)

	ip := types.NetProtocolCounters{
		Protocol: "IPv4",
	}

	for i, header := range headers {
		if i >= len(values) {
			continue
		}
		value, _ := strconv.ParseUint(values[i], 10, 64)
		
		switch header {
		case "InReceives":
			ip.PacketsRecv = value
		case "OutRequests":
			ip.PacketsSent = value
		case "InHdrErrors":
		case "InAddrErrors":
		case "OutDiscards":
			ip.Dropped = value
		case "InDelivers":
		}
	}

	return ip, nil
}

// getInterfaceSpeed gets the speed of a network interface in bits per second
func getInterfaceSpeed(ifaceName string) uint64 {
	// Try to read from sysfs
	speedPath := fmt.Sprintf("/sys/class/net/%s/speed", ifaceName)
	content, err := os.ReadFile(speedPath)
	if err != nil {
		return 0
	}

	speed, err := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
	if err != nil {
		return 0
	}

	// Speed is in Mbps, convert to bps
	return speed * 1000 * 1000
}

// getDefaultGateway gets the default gateway
func getDefaultGateway() string {
	content, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	
	// Skip header
	scanner.Scan()

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 8 && fields[1] == "00000000" {
			// Found default route, get gateway
			gwHex := fields[2]
			gwIP := hexToIP(gwHex)
			if gwIP != "" {
				return gwIP
			}
		}
	}

	return ""
}

// getDNServers gets DNS servers from resolv.conf
func getDNServers() []string {
	content, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}

	var dns []string
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				dns = append(dns, parts[1])
			}
		}
	}

	return dns
}

// hexToIP converts hex-encoded IP to dotted notation
func hexToIP(hexIP string) string {
	if len(hexIP) != 8 {
		return ""
	}

	bytes, err := hex.DecodeString(hexIP)
	if err != nil {
		return ""
	}

	// Linux stores IP in little-endian format
	return fmt.Sprintf("%d.%d.%d.%d", bytes[3], bytes[2], bytes[1], bytes[0])
}

// flagsToStrings converts net.Flags to string slice
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
