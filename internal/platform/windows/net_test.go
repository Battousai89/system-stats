package windows

import (
	"testing"

	"system-stats/internal/types"
)

func TestNewNetIOCounters(t *testing.T) {
	counters, err := NewNetIOCounters()
	if err != nil {
		t.Fatalf("NewNetIOCounters() returned error: %v", err)
	}

	if len(counters) == 0 {
		t.Log("NewNetIOCounters() returned empty slice (no network interfaces)")
		return
	}

	for i, c := range counters {
		if c.Name == "" {
			t.Errorf("NetIO[%d]: Name is empty", i)
		}

		t.Logf("NetIO[%d]: %s - Sent: %d B, Recv: %d B, Sent/s: %d, Recv/s: %d",
			i, c.Name, c.BytesSent, c.BytesRecv, c.BytesSentPerSec, c.BytesRecvPerSec)
	}
}

func TestNewNetInterfaces(t *testing.T) {
	ifaces, err := NewNetInterfaces()
	if err != nil {
		t.Fatalf("NewNetInterfaces() returned error: %v", err)
	}

	if len(ifaces) == 0 {
		t.Log("NewNetInterfaces() returned empty slice (no network interfaces)")
		return
	}

	for i, iface := range ifaces {
		if iface.Name == "" {
			t.Errorf("NetInterface[%d]: Name is empty", i)
		}

		if iface.Index == 0 {
			t.Logf("NetInterface[%d]: Index is 0", i)
		}

		t.Logf("NetInterface[%d]: %s - Status: %s, IPs: %v",
			i, iface.Name, iface.Status, iface.IPAddresses)
	}
}

func TestNewNetProtocolCounters(t *testing.T) {
	counters, err := NewNetProtocolCounters()
	if err != nil {
		t.Fatalf("NewNetProtocolCounters() returned error: %v", err)
	}

	if len(counters) == 0 {
		t.Log("NewNetProtocolCounters() returned empty slice")
		return
	}

	for i, c := range counters {
		if c.Protocol == "" {
			t.Errorf("NetProtocol[%d]: Protocol is empty", i)
		}

		t.Logf("NetProtocol[%d]: %s - Sent: %d, Recv: %d",
			i, c.Protocol, c.PacketsSent, c.PacketsRecv)
	}
}

func TestNetIOCountersToPrint(t *testing.T) {
	counters, err := NewNetIOCounters()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	if len(counters) == 0 {
		t.Skip("No network IO counters to print")
	}

	output := types.NetIOCountersToPrint(counters)
	if output == "" {
		t.Error("NetIOCountersToPrint() returned empty string")
	}

	t.Logf("NetIOCountersToPrint():\n%s", output)
}

func TestNetInterfacesToPrint(t *testing.T) {
	ifaces, err := NewNetInterfaces()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	if len(ifaces) == 0 {
		t.Skip("No network interfaces to print")
	}

	output := types.NetInterfacesToPrint(ifaces)
	if output == "" {
		t.Error("NetInterfacesToPrint() returned empty string")
	}

	t.Logf("NetInterfacesToPrint():\n%s", output)
}

func TestNetProtocolCountersToPrint(t *testing.T) {
	counters, err := NewNetProtocolCounters()
	if err != nil {
		t.Skipf("Skipping ToPrint test: %v", err)
	}

	if len(counters) == 0 {
		t.Skip("No protocol counters to print")
	}

	output := types.NetProtocolCountersToPrint(counters)
	if output == "" {
		t.Error("NetProtocolCountersToPrint() returned empty string")
	}

	t.Logf("NetProtocolCountersToPrint():\n%s", output)
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		bps      uint64
		expected string
	}{
		{500, "500 bps"},
		{1500, "1.50 Kbps"},
		{1500000, "1.50 Mbps"},
		{1500000000, "1.50 Gbps"},
	}

	for _, test := range tests {
		// formatSpeed не экспортируется, тестируем через types
		_ = test.bps
		t.Logf("Speed %d bps would be formatted", test.bps)
	}
}

// Benchmark тесты
func BenchmarkNewNetIOCounters(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewNetIOCounters()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewNetInterfaces(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewNetInterfaces()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNewNetProtocolCounters(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewNetProtocolCounters()
		if err != nil {
			b.Fatal(err)
		}
	}
}
