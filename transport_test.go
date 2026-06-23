package fins

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTransport_NewTransport(t *testing.T) {
	cfg := map[string]interface{}{
		"plcIP":   "127.0.0.1",
		"plcPort": 9600,
		"timeout": 5000,
	}

	transport, err := NewTransport(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transport.plcIP != "127.0.0.1" {
		t.Errorf("expected plcIP 127.0.0.1, got %s", transport.plcIP)
	}
	if transport.plcPort != 9600 {
		t.Errorf("expected plcPort 9600, got %d", transport.plcPort)
	}
	if transport.timeout != 5000*time.Millisecond {
		t.Errorf("expected timeout 5s, got %v", transport.timeout)
	}
}

func TestTransport_NewTransport_MissingIP(t *testing.T) {
	cfg := map[string]interface{}{
		"plcPort": 9600,
	}

	_, err := NewTransport(cfg)
	if err == nil {
		t.Fatal("expected error for missing plcIP")
	}
}

func TestTransport_ConnectDisconnect(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	cfg := map[string]interface{}{
		"plcIP":             "127.0.0.1",
		"plcPort":           0,
		"timeout":           2000,
		"heartbeatInterval": 0,
	}

	_, port, _ := netSplitHostPort(addr)
	cfg["plcPort"] = port

	transport, err := NewTransport(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	if !transport.IsConnected() {
		t.Error("expected connected")
	}

	if err := transport.Disconnect(); err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}

	if transport.IsConnected() {
		t.Error("expected disconnected")
	}
}

func TestTransport_ReadMemory(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	mockPLC.SetWord(MemoryAreaDMWord, 100, 1234)
	mockPLC.SetWord(MemoryAreaDMWord, 101, 5678)

	cfg := map[string]interface{}{
		"plcIP":             "127.0.0.1",
		"plcPort":           0,
		"timeout":           2000,
		"heartbeatInterval": 0,
		"srcNodeAddr":       2,
		"dstNodeAddr":       1,
	}

	_, port, _ := netSplitHostPort(addr)
	cfg["plcPort"] = port

	transport, err := NewTransport(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	data, err := transport.ReadMemory(ctx, MemoryAreaDMWord, 100, 2)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if len(data) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(data))
	}
}

func TestTransport_WriteMemory(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	cfg := map[string]interface{}{
		"plcIP":             "127.0.0.1",
		"plcPort":           0,
		"timeout":           2000,
		"heartbeatInterval": 0,
		"srcNodeAddr":       2,
		"dstNodeAddr":       1,
	}

	_, port, _ := netSplitHostPort(addr)
	cfg["plcPort"] = port

	transport, err := NewTransport(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	writeData := []byte{0x04, 0xD2, 0x16, 0x2E}
	err = transport.WriteMemory(ctx, MemoryAreaDMWord, 200, 2, writeData)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}

	val := mockPLC.GetWord(MemoryAreaDMWord, 200)
	if val != 1234 {
		t.Errorf("expected word 200 = 1234, got %d", val)
	}

	val2 := mockPLC.GetWord(MemoryAreaDMWord, 201)
	if val2 != 5678 {
		t.Errorf("expected word 201 = 5678, got %d", val2)
	}
}

func TestTransport_SendCommand_NotConnected(t *testing.T) {
	cfg := map[string]interface{}{
		"plcIP":   "127.0.0.1",
		"plcPort": 9999,
		"timeout": 500,
	}

	transport, err := NewTransport(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	_, err = transport.SendCommand(ctx, []byte{0x01, 0x01})
	if err == nil {
		t.Fatal("expected error when not connected")
	}
	if err != ErrNotConnected {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestTransport_GetMetrics(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	cfg := map[string]interface{}{
		"plcIP":             "127.0.0.1",
		"plcPort":           0,
		"timeout":           2000,
		"heartbeatInterval": 0,
	}

	_, port, _ := netSplitHostPort(addr)
	cfg["plcPort"] = port

	transport, err := NewTransport(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	metrics := transport.GetMetrics()
	if metrics.Connected {
		t.Error("expected not connected before Connect")
	}

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	metrics = transport.GetMetrics()
	if !metrics.Connected {
		t.Error("expected connected after Connect")
	}
	if metrics.RemoteAddr == "" {
		t.Error("expected non-empty remote address")
	}
}

func TestTransport_SetTimeout(t *testing.T) {
	cfg := map[string]interface{}{
		"plcIP":   "127.0.0.1",
		"plcPort": 9600,
		"timeout": 3000,
	}

	transport, err := NewTransport(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transport.GetTimeout() != 3000*time.Millisecond {
		t.Errorf("expected timeout 3s, got %v", transport.GetTimeout())
	}

	transport.SetTimeout(5000 * time.Millisecond)
	if transport.GetTimeout() != 5000*time.Millisecond {
		t.Errorf("expected timeout 5s, got %v", transport.GetTimeout())
	}
}

func TestTransport_AddressConfig(t *testing.T) {
	cfg := map[string]interface{}{
		"plcIP":          "127.0.0.1",
		"plcPort":        9600,
		"srcNetworkAddr": 1,
		"srcNodeAddr":    10,
		"srcUnitAddr":    0,
		"dstNetworkAddr": 2,
		"dstNodeAddr":    20,
		"dstUnitAddr":    1,
	}

	transport, err := NewTransport(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src := transport.SrcAddress()
	if src.Network != 1 || src.Node != 10 || src.Unit != 0 {
		t.Errorf("unexpected src address: %+v", src)
	}

	dst := transport.DstAddress()
	if dst.Network != 2 || dst.Node != 20 || dst.Unit != 1 {
		t.Errorf("unexpected dst address: %+v", dst)
	}
}

func TestTransport_WriteReadBits(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, err := NewTransport(map[string]interface{}{
		"plcIP":             "127.0.0.1",
		"plcPort":           0,
		"timeout":           2000,
		"heartbeatInterval": 0,
	})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	_, port, _ := netSplitHostPort(addr)
	transport.plcPort = port

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	bits := []bool{true, false, true, true}
	err = transport.WriteBits(ctx, MemoryAreaCIOBit, 0, 0, bits)
	if err != nil {
		t.Fatalf("write bits failed: %v", err)
	}

	readBits, err := transport.ReadBits(ctx, MemoryAreaCIOBit, 0, 0, 4)
	if err != nil {
		t.Fatalf("read bits failed: %v", err)
	}

	if len(readBits) != len(bits) {
		t.Fatalf("expected %d bits, got %d", len(bits), len(readBits))
	}

	for i, b := range bits {
		if readBits[i] != b {
			t.Errorf("bit %d: expected %v, got %v", i, b, readBits[i])
		}
	}
}

func TestTransport_Reconnect(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, err := NewTransport(map[string]interface{}{
		"plcIP":             "127.0.0.1",
		"plcPort":           0,
		"timeout":           2000,
		"heartbeatInterval": 0,
	})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}
	_, port, _ := netSplitHostPort(addr)
	transport.plcPort = port

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	if !transport.IsConnected() {
		t.Fatal("expected connected")
	}

	if err := transport.Reconnect(ctx); err != nil {
		t.Fatalf("reconnect failed: %v", err)
	}

	if !transport.IsConnected() {
		t.Error("expected connected after reconnect")
	}
}

func TestTransport_GetTimeout(t *testing.T) {
	transport, err := NewTransport(map[string]interface{}{
		"plcIP":   "127.0.0.1",
		"plcPort": 9600,
		"timeout": 5000,
	})
	if err != nil {
		t.Fatalf("failed to create transport: %v", err)
	}

	timeout := transport.GetTimeout()
	if timeout != 5000*time.Millisecond {
		t.Errorf("expected 5000ms timeout, got %v", timeout)
	}
}

func netSplitHostPort(addr string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, err
	}
	port := 0
	for _, c := range portStr {
		port = port*10 + int(c-'0')
	}
	return host, port, nil
}
