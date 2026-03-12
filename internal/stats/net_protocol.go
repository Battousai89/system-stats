package stats

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"system-stats/internal/formatter"
)

type NetProtocolCounters struct {
	Protocol string          `json:"protocol"`
	Stats    map[string]int64 `json:"stats"`
}

type NetConntrackStats struct {
	Entries      uint64 `json:"entries"`
	Searched     uint64 `json:"searched"`
	Found        uint64 `json:"found"`
	New          uint64 `json:"new"`
	Invalid      uint64 `json:"invalid"`
	Ignore       uint64 `json:"ignore"`
	Delete       uint64 `json:"delete"`
	Insert       uint64 `json:"insert"`
	Drop         uint64 `json:"drop"`
	EarlyDrop    uint64 `json:"earlyDrop"`
	IcmpError    uint64 `json:"icmpError"`
	ExpectNew    uint64 `json:"expectNew"`
	ExpectCreate uint64 `json:"expectCreate"`
	ExpectDelete uint64 `json:"expectDelete"`
}

type NetFilterCounters struct {
	ConnTrackCount int64 `json:"connTrackCount"`
	ConnTrackMax   int64 `json:"connTrackMax"`
}

func NewNetProtocolCounters() ([]NetProtocolCounters, error) {
	switch runtime.GOOS {
	case "linux":
		return parseProcNetSnmp()
	case "windows":
		return getWindowsProtocolCounters()
	case "darwin", "freebsd":
		return getUnixProtocolCounters()
	default:
		return []NetProtocolCounters{}, nil
	}
}

func parseProcNetSnmp() ([]NetProtocolCounters, error) {
	var result []NetProtocolCounters

	tcpStats, err := parseProtocolStats("/proc/net/snmp", "Tcp")
	if err == nil && len(tcpStats) > 0 {
		result = append(result, NetProtocolCounters{
			Protocol: "TCP",
			Stats:    tcpStats,
		})
	}

	udpStats, err := parseProtocolStats("/proc/net/snmp", "Udp")
	if err == nil && len(udpStats) > 0 {
		result = append(result, NetProtocolCounters{
			Protocol: "UDP",
			Stats:    udpStats,
		})
	}

	ipStats, err := parseProtocolStats("/proc/net/snmp", "Ip")
	if err == nil && len(ipStats) > 0 {
		result = append(result, NetProtocolCounters{
			Protocol: "IP",
			Stats:    ipStats,
		})
	}

	if len(result) == 0 {
		return []NetProtocolCounters{}, nil
	}

	return result, nil
}

func parseProtocolStats(path string, protocol string) (map[string]int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var headers []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 2 {
			continue
		}

		fieldName := strings.TrimSuffix(fields[0], ":")

		if fieldName == protocol {
			if len(fields) > 1 && !isNumeric(fields[1]) {
				headers = fields[1:]
			} else if len(headers) > 0 {
				stats := make(map[string]int64)
				for i := 1; i < len(fields) && i-1 < len(headers); i++ {
					val, _ := strconv.ParseInt(fields[i], 10, 64)
					stats[headers[i-1]] = val
				}
				return stats, nil
			}
		}
	}

	return nil, nil
}

func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	start := 0
	if s[0] == '-' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	for _, c := range s[start:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func getWindowsProtocolCounters() ([]NetProtocolCounters, error) {
	output, err := runCommandWithTimeout("netstat", "-s")
	if err != nil {
		return []NetProtocolCounters{}, nil
	}

	var result []NetProtocolCounters
	lines := strings.Split(string(output), "\n")
	currentProto := ""
	currentStats := make(map[string]int64)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "IPv4") || strings.Contains(line, "TCP") || strings.Contains(line, "UDP") {
			if currentProto != "" && len(currentStats) > 0 {
				result = append(result, NetProtocolCounters{
					Protocol: currentProto,
					Stats:    currentStats,
				})
			}
			currentProto = strings.TrimRight(line, ":")
			currentStats = make(map[string]int64)
			continue
		}

		if strings.Contains(line, ":") && currentProto != "" {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
				currentStats[key] = val
			}
		}
	}

	if currentProto != "" && len(currentStats) > 0 {
		result = append(result, NetProtocolCounters{
			Protocol: currentProto,
			Stats:    currentStats,
		})
	}

	return result, nil
}

func getUnixProtocolCounters() ([]NetProtocolCounters, error) {
	output, err := runCommandWithTimeout("netstat", "-s")
	if err != nil {
		return []NetProtocolCounters{}, nil
	}

	var result []NetProtocolCounters
	lines := strings.Split(string(output), "\n")
	currentProto := ""
	currentStats := make(map[string]int64)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasSuffix(line, ":") && len(line) < 10 {
			if currentProto != "" && len(currentStats) > 0 {
				result = append(result, NetProtocolCounters{
					Protocol: currentProto,
					Stats:    currentStats,
				})
			}
			currentProto = strings.TrimRight(line, ":")
			currentStats = make(map[string]int64)
			continue
		}

		if strings.Contains(line, "=") && currentProto != "" {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
				currentStats[key] = val
			}
		}
	}

	if currentProto != "" && len(currentStats) > 0 {
		result = append(result, NetProtocolCounters{
			Protocol: currentProto,
			Stats:    currentStats,
		})
	}

	return result, nil
}

func NewNetConntrackStats() (*NetConntrackStats, error) {
	stats := &NetConntrackStats{}

	switch runtime.GOOS {
	case "linux":
		file, err := os.Open("/proc/net/stat/nf_conntrack")
		if err != nil {
			return stats, nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 17 {
				continue
			}

			if fields[0] == "entries" {
				continue
			}

			stats.Entries, _ = strconv.ParseUint(fields[0], 16, 64)
			stats.Searched, _ = strconv.ParseUint(fields[1], 16, 64)
			stats.Found, _ = strconv.ParseUint(fields[2], 16, 64)
			stats.New, _ = strconv.ParseUint(fields[3], 16, 64)
			stats.Invalid, _ = strconv.ParseUint(fields[4], 16, 64)
			stats.Ignore, _ = strconv.ParseUint(fields[5], 16, 64)
			stats.Delete, _ = strconv.ParseUint(fields[6], 16, 64)
			stats.Insert, _ = strconv.ParseUint(fields[7], 16, 64)
			stats.Drop, _ = strconv.ParseUint(fields[8], 16, 64)
			stats.EarlyDrop, _ = strconv.ParseUint(fields[9], 16, 64)
			stats.IcmpError, _ = strconv.ParseUint(fields[10], 16, 64)
		}
	}

	return stats, nil
}

func NewNetFilterCounters() (*NetFilterCounters, error) {
	counters := &NetFilterCounters{}

	switch runtime.GOOS {
	case "linux":
		if data, err := os.ReadFile("/proc/sys/net/netfilter/nf_conntrack_count"); err == nil {
			counters.ConnTrackCount, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}

		if data, err := os.ReadFile("/proc/sys/net/netfilter/nf_conntrack_max"); err == nil {
			counters.ConnTrackMax, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		}
	}

	return counters, nil
}

func (c NetProtocolCounters) ToPrint() string {
	b := formatter.NewBuilder().
		AddField("Protocol", c.Protocol, "")

	for key, value := range c.Stats {
		b.AddField(key, value, "")
	}

	return b.Build()
}

func (s NetConntrackStats) ToPrint() string {
	return formatter.NewBuilder().
		AddField("Entries", s.Entries, "").
		AddField("Searched", s.Searched, "").
		AddField("Found", s.Found, "").
		AddField("New", s.New, "").
		AddField("Invalid", s.Invalid, "").
		AddField("Ignore", s.Ignore, "").
		AddField("Delete", s.Delete, "").
		AddField("Insert", s.Insert, "").
		AddField("Drop", s.Drop, "").
		AddField("EarlyDrop", s.EarlyDrop, "").
		AddField("IcmpError", s.IcmpError, "").
		Build()
}

func (f NetFilterCounters) ToPrint() string {
	return formatter.NewBuilder().
		AddField("ConnTrackCount", f.ConnTrackCount, "").
		AddField("ConnTrackMax", f.ConnTrackMax, "").
		Build()
}

func NetProtocolCountersToPrint(counters []NetProtocolCounters) string {
	var sb strings.Builder
	for i, c := range counters {
		sb.WriteString(fmt.Sprintf("  %s:\n", c.Protocol))
		sb.WriteString(c.ToPrint())
		if i < len(counters)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
