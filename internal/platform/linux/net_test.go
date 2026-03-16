//go:build linux
// +build linux

package linux

import (
	"testing"
)

func TestNewNetIOCounters(t *testing.T) {
	counters, err := NewNetIOCounters()
	if err != nil {
		t.Fatalf("NewNetIOCounters() returned error: %v", err)
	}

	if counters == nil {
		t.Fatal("NewNetIOCounters() returned nil")
	}

	if len(counters) == 0 {
		t.Log("No network interfaces found")
		return
	}

	for i, c := range counters {
		if c.Name == "" {
			t.Errorf("NetIO[%d]: Name is empty", i)
		}

		t.Logf("NetIO[%d]: %s - BytesRecv=%d, BytesSent=%d",
			i, c.Name, c.BytesRecv, c.BytesSent)
	}
}

func TestNewNetInterfaces(t *testing.T) {
	ifaces, err := NewNetInterfaces()
	if err != nil {
		t.Fatalf("NewNetInterfaces() returned error: %v", err)
	}

	if ifaces == nil {
		t.Fatal("NewNetInterfaces() returned nil")
	}

	if len(ifaces) == 0 {
		t.Log("No network interfaces found")
		return
	}

	for i, iface := range ifaces {
		if iface.Name == "" {
			t.Errorf("NetInterface[%d]: Name is empty", i)
		}

		t.Logf("NetInterface[%d]: %s - Status=%s, IP=%v",
			i, iface.Name, iface.Status, iface.IPAddresses)
	}
}

func TestNewNetProtocolCounters(t *testing.T) {
	counters, err := NewNetProtocolCounters()
	if err != nil {
		t.Fatalf("NewNetProtocolCounters() returned error: %v", err)
	}

	if counters == nil {
		t.Fatal("NewNetProtocolCounters() returned nil")
	}

	if len(counters) == 0 {
		t.Log("No protocol counters found")
		return
	}

	for i, c := range counters {
		if c.Protocol == "" {
			t.Errorf("NetProto[%d]: Protocol is empty", i)
		}

		t.Logf("NetProto[%d]: %s - PacketsSent=%d, PacketsRecv=%d",
			i, c.Protocol, c.PacketsSent, c.PacketsRecv)
	}
}

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
