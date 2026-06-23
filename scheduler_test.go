package fins

import (
	"context"
	"testing"
	"time"
)

func TestScheduler_NewScheduler(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)

	scheduler := NewScheduler(transport, decoder, nil)
	if scheduler == nil {
		t.Fatal("expected scheduler, got nil")
	}

	if scheduler.MaxFrameLength() <= 0 {
		t.Errorf("expected positive max frame length, got %d", scheduler.MaxFrameLength())
	}
}

func TestScheduler_ReadPoints_Word(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	mockPLC.SetWord(MemoryAreaDMWord, 100, 1234)
	mockPLC.SetWord(MemoryAreaDMWord, 101, 5678)
	mockPLC.SetWord(MemoryAreaDMWord, 200, 9999)

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	points := []Point{
		{ID: "D100", Address: "D100", DataType: DataTypeUINT16},
		{ID: "D101", Address: "D101", DataType: DataTypeUINT16},
		{ID: "D200", Address: "D200", DataType: DataTypeUINT16},
	}

	values, err := scheduler.ReadPoints(ctx, points)
	if err != nil {
		t.Fatalf("ReadPoints failed: %v", err)
	}

	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}

	if v, ok := values["D100"]; !ok || v.Value != uint16(1234) {
		t.Errorf("expected D100=1234, got %v", v)
	}
	if v, ok := values["D101"]; !ok || v.Value != uint16(5678) {
		t.Errorf("expected D101=5678, got %v", v)
	}
	if v, ok := values["D200"]; !ok || v.Value != uint16(9999) {
		t.Errorf("expected D200=9999, got %v", v)
	}

	stats := scheduler.GetStats()
	if stats.TotalRequests == 0 {
		t.Error("expected total requests > 0")
	}
}

func TestScheduler_ReadPoints_Bit(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	mockPLC.SetWord(MemoryAreaCIOWord, 0, 0x0001)
	mockPLC.SetWord(MemoryAreaCIOWord, 10, 0x8000)

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	points := []Point{
		{ID: "CIO0.0", Address: "CIO0.0", DataType: DataTypeBIT},
		{ID: "CIO0.15", Address: "CIO0.15", DataType: DataTypeBIT},
		{ID: "CIO10.15", Address: "CIO10.15", DataType: DataTypeBIT},
	}

	values, err := scheduler.ReadPoints(ctx, points)
	if err != nil {
		t.Fatalf("ReadPoints failed: %v", err)
	}

	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}

	if v, ok := values["CIO0.0"]; !ok || v.Value != true {
		t.Errorf("expected CIO0.0=true, got %v", v)
	}
	if v, ok := values["CIO0.15"]; !ok || v.Value != false {
		t.Errorf("expected CIO0.15=false, got %v", v)
	}
	if v, ok := values["CIO10.15"]; !ok || v.Value != true {
		t.Errorf("expected CIO10.15=true, got %v", v)
	}
}

func TestScheduler_WritePoint(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	point := Point{ID: "D50", Address: "D50", DataType: DataTypeUINT16}
	err = scheduler.WritePoint(ctx, point, uint16(4321))
	if err != nil {
		t.Fatalf("WritePoint failed: %v", err)
	}

	wordVal := mockPLC.GetWord(MemoryAreaDMWord, 50)
	if wordVal != 4321 {
		t.Errorf("expected D50=4321, got %d", wordVal)
	}
}

func TestScheduler_WritePoint_Bit(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	point := Point{ID: "CIO0.5", Address: "CIO0.5", DataType: DataTypeBIT}
	err = scheduler.WritePoint(ctx, point, true)
	if err != nil {
		t.Fatalf("WritePoint bit failed: %v", err)
	}

	wordVal := mockPLC.GetWord(MemoryAreaCIOWord, 0)
	if wordVal&(1<<5) == 0 {
		t.Errorf("expected CIO0.5=true, but bit not set, word=%04x", wordVal)
	}

	err = scheduler.WritePoint(ctx, point, false)
	if err != nil {
		t.Fatalf("WritePoint bit failed: %v", err)
	}

	wordVal = mockPLC.GetWord(MemoryAreaCIOWord, 0)
	if wordVal&(1<<5) != 0 {
		t.Errorf("expected CIO0.5=false, but bit is set, word=%04x", wordVal)
	}
}

func TestScheduler_WritePoint_String(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	point := Point{ID: "D200", Address: "D200.10L", DataType: DataTypeSTRING}
	testStr := "Hello"
	err = scheduler.WritePoint(ctx, point, testStr)
	if err != nil {
		t.Fatalf("WritePoint string failed: %v", err)
	}

	readPoints := []Point{{ID: "D200", Address: "D200.10L", DataType: DataTypeSTRING}}
	values, err := scheduler.ReadPoints(ctx, readPoints)
	if err != nil {
		t.Fatalf("ReadPoints string failed: %v", err)
	}

	if v, ok := values["D200"]; !ok || v.Value != testStr {
		t.Errorf("expected D200=%q, got %v", testStr, v.Value)
	}
}

func TestScheduler_WritePoint_INT32(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	point := Point{ID: "D300", Address: "D300", DataType: DataTypeINT32}
	testVal := int32(-123456)
	err = scheduler.WritePoint(ctx, point, testVal)
	if err != nil {
		t.Fatalf("WritePoint INT32 failed: %v", err)
	}

	readPoints := []Point{{ID: "D300", Address: "D300", DataType: DataTypeINT32}}
	values, err := scheduler.ReadPoints(ctx, readPoints)
	if err != nil {
		t.Fatalf("ReadPoints INT32 failed: %v", err)
	}

	if v, ok := values["D300"]; !ok || v.Value != testVal {
		t.Errorf("expected D300=%d, got %v", testVal, v.Value)
	}
}

func TestScheduler_WritePoint_InvalidAddress(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	point := Point{ID: "invalid", Address: "INVALID", DataType: DataTypeUINT16}
	err = scheduler.WritePoint(ctx, point, uint16(100))
	if err == nil {
		t.Error("expected error for invalid address")
	}
}

func TestScheduler_WritePoint_NotConnected(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	point := Point{ID: "D10", Address: "D10", DataType: DataTypeUINT16}
	err = scheduler.WritePoint(ctx, point, uint16(100))
	if err == nil {
		t.Error("expected error when not connected")
	}
}

func TestScheduler_MixedDataTypes(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	mockPLC.SetWord(MemoryAreaDMWord, 0, 0x1234)
	mockPLC.SetWord(MemoryAreaDMWord, 1, 0x5678)

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	points := []Point{
		{ID: "val_uint16", Address: "D0", DataType: DataTypeUINT16},
		{ID: "val_uint32", Address: "D0", DataType: DataTypeUINT32},
		{ID: "val_int16", Address: "D1", DataType: DataTypeINT16},
	}

	values, err := scheduler.ReadPoints(ctx, points)
	if err != nil {
		t.Fatalf("ReadPoints failed: %v", err)
	}

	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}

	if _, ok := values["val_uint16"]; !ok {
		t.Error("missing val_uint16")
	}
	if _, ok := values["val_uint32"]; !ok {
		t.Error("missing val_uint32")
	}
	if _, ok := values["val_int16"]; !ok {
		t.Error("missing val_int16")
	}
}

func TestScheduler_EmptyPoints(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	values, err := scheduler.ReadPoints(ctx, []Point{})
	if err != nil {
		t.Fatalf("ReadPoints with empty points failed: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("expected empty result, got %d values", len(values))
	}
}

func TestScheduler_Stats(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	mockPLC.SetWord(MemoryAreaDMWord, 10, 100)

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	scheduler.ResetStats()
	stats := scheduler.GetStats()
	if stats.TotalRequests != 0 {
		t.Errorf("expected 0 total requests after reset, got %d", stats.TotalRequests)
	}

	points := []Point{
		{ID: "D10", Address: "D10", DataType: DataTypeUINT16},
	}
	_, err = scheduler.ReadPoints(ctx, points)
	if err != nil {
		t.Fatalf("ReadPoints failed: %v", err)
	}

	stats = scheduler.GetStats()
	if stats.TotalRequests != 1 {
		t.Errorf("expected 1 total request, got %d", stats.TotalRequests)
	}
	if stats.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", stats.SuccessCount)
	}
}

func TestScheduler_Health(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	health := scheduler.Health()
	if health != HealthStatusDown {
		t.Errorf("expected status %s, got %s", HealthStatusDown, health)
	}

	ctx := context.Background()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer transport.Disconnect()

	health = scheduler.Health()
	if health != HealthStatusUp {
		t.Errorf("expected status %s, got %s", HealthStatusUp, health)
	}
}

func TestScheduler_SetMaxFrameLength(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	scheduler.SetMaxFrameLength(500)
	if scheduler.MaxFrameLength() != 500 {
		t.Errorf("expected max frame length 500, got %d", scheduler.MaxFrameLength())
	}
}

func TestScheduler_SetMinInterval(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	scheduler.SetMinInterval(10 * time.Millisecond)
	time.Sleep(5 * time.Millisecond)
}

func TestScheduler_String(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	str := scheduler.String()
	if str == "" {
		t.Error("expected non-empty string description")
	}
}

func TestScheduler_NotConnected(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	transport, decoder := setupTestTransport(addr)
	scheduler := NewScheduler(transport, decoder, nil)

	ctx := context.Background()
	points := []Point{
		{ID: "D10", Address: "D10", DataType: DataTypeUINT16},
	}

	results, err := scheduler.ReadPoints(ctx, points)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for id, val := range results {
		if val.Quality != QualityBad {
			t.Errorf("expected bad quality for %s, got %s", id, val.Quality)
		}
	}
}

func setupTestTransport(addr string) (*Transport, *Decoder) {
	transport, _ := NewTransport(map[string]interface{}{
		"plcIP":             "127.0.0.1",
		"plcPort":           0,
		"timeout":           2000,
		"heartbeatInterval": 0,
	})
	_, port, _ := netSplitHostPort(addr)
	transport.plcPort = port
	decoder := NewDecoder()
	return transport, decoder
}
