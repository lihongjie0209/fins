package fins

import (
	"context"
	"testing"
)

func TestFinsTCPDriver_NewDriver(t *testing.T) {
	driver := NewFinsTCPDriver()
	if driver == nil {
		t.Fatal("expected non-nil driver")
	}
	if driver.IsInitialized() {
		t.Error("expected driver not initialized")
	}
	if driver.IsConnected() {
		t.Error("expected driver not connected")
	}
}

func TestFinsTCPDriver_Init(t *testing.T) {
	driver := NewFinsTCPDriver()

	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": 9600,
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if !driver.IsInitialized() {
		t.Error("expected driver initialized")
	}

	if err := driver.Init(cfg); err == nil {
		t.Error("expected error on double init")
	}
}

func TestFinsTCPDriver_Init_MissingConfig(t *testing.T) {
	driver := NewFinsTCPDriver()

	cfg := DriverConfig{
		Config: map[string]interface{}{},
	}

	err := driver.Init(cfg)
	if err == nil {
		t.Error("expected error with missing config")
	}
}

func TestFinsTCPDriver_ConnectDisconnect(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	if !driver.IsConnected() {
		t.Error("expected driver connected")
	}

	if err := driver.Disconnect(); err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}

	if driver.IsConnected() {
		t.Error("expected driver disconnected")
	}
}

func TestFinsTCPDriver_Connect_NotInitialized(t *testing.T) {
	driver := NewFinsTCPDriver()
	ctx := context.Background()
	err := driver.Connect(ctx)
	if err == nil {
		t.Error("expected error when not initialized")
	}
}

func TestFinsTCPDriver_ReadPoints(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	mockPLC.SetWord(MemoryAreaDMWord, 100, 1234)
	mockPLC.SetWord(MemoryAreaDMWord, 101, 5678)
	mockPLC.SetWord(MemoryAreaCIOWord, 0, 0x0001)

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	points := []Point{
		{ID: "D100", Address: "D100", DataType: DataTypeUINT16},
		{ID: "D101", Address: "D101", DataType: DataTypeUINT16},
		{ID: "CIO0.0", Address: "CIO0.0", DataType: DataTypeBIT},
	}

	results, err := driver.ReadPoints(ctx, points)
	if err != nil {
		t.Fatalf("read points failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results["D100"].Value != uint16(1234) {
		t.Errorf("expected D100=1234, got %v", results["D100"].Value)
	}
	if results["D100"].Quality != QualityGood {
		t.Errorf("expected D100 quality good, got %s", results["D100"].Quality)
	}

	if results["D101"].Value != uint16(5678) {
		t.Errorf("expected D101=5678, got %v", results["D101"].Value)
	}

	if results["CIO0.0"].Value != true {
		t.Errorf("expected CIO0.0=true, got %v", results["CIO0.0"].Value)
	}
}

func TestFinsTCPDriver_ReadPoints_NotInitialized(t *testing.T) {
	driver := NewFinsTCPDriver()
	ctx := context.Background()
	points := []Point{{ID: "D100", Address: "D100", DataType: DataTypeUINT16}}
	_, err := driver.ReadPoints(ctx, points)
	if err == nil {
		t.Error("expected error when not initialized")
	}
}

func TestFinsTCPDriver_WritePoint(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	point := Point{ID: "D200", Address: "D200", DataType: DataTypeUINT16}
	if err := driver.WritePoint(ctx, point, uint16(4321)); err != nil {
		t.Fatalf("write point failed: %v", err)
	}

	val := mockPLC.GetWord(MemoryAreaDMWord, 200)
	if val != 4321 {
		t.Errorf("expected D200=4321, got %d", val)
	}
}

func TestFinsTCPDriver_WritePoint_NotInitialized(t *testing.T) {
	driver := NewFinsTCPDriver()
	ctx := context.Background()
	point := Point{ID: "D100", Address: "D100", DataType: DataTypeUINT16}
	err := driver.WritePoint(ctx, point, uint16(123))
	if err == nil {
		t.Error("expected error when not initialized")
	}
}

func TestFinsTCPDriver_Health(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	health := driver.Health()
	if health != HealthStatusUnknown {
		t.Errorf("expected unknown health when not initialized, got %s", health)
	}

	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	health = driver.Health()
	if health != HealthStatusDown {
		t.Errorf("expected down health when not connected, got %s", health)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	health = driver.Health()
	if health != HealthStatusUp {
		t.Errorf("expected up health when connected, got %s", health)
	}
}

func TestFinsTCPDriver_SetSlaveID(t *testing.T) {
	driver := NewFinsTCPDriver()
	if err := driver.SetSlaveID(10); err != nil {
		t.Fatalf("set slave id failed: %v", err)
	}
}

func TestFinsTCPDriver_SetDeviceConfig(t *testing.T) {
	driver := NewFinsTCPDriver()
	config := map[string]interface{}{
		"timeout": 5000,
	}
	if err := driver.SetDeviceConfig(config); err != nil {
		t.Fatalf("set device config failed: %v", err)
	}
}

func TestFinsTCPDriver_GetConnectionMetrics(t *testing.T) {
	driver := NewFinsTCPDriver()
	metrics := driver.GetConnectionMetrics()
	if metrics.Connected != false {
		t.Errorf("expected not connected when not initialized")
	}
}

func TestFinsTCPDriver_GetSchedulerStats(t *testing.T) {
	driver := NewFinsTCPDriver()
	stats := driver.GetSchedulerStats()
	if stats.TotalRequests != 0 {
		t.Errorf("expected 0 total requests when not initialized, got %d", stats.TotalRequests)
	}
}

func TestFinsTCPDriver_Reconnect(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	if !driver.IsConnected() {
		t.Fatal("expected connected")
	}

	if err := driver.Reconnect(ctx); err != nil {
		t.Fatalf("reconnect failed: %v", err)
	}

	if !driver.IsConnected() {
		t.Error("expected connected after reconnect")
	}
}

func TestFinsTCPDriver_Reconnect_NotInitialized(t *testing.T) {
	driver := NewFinsTCPDriver()
	ctx := context.Background()
	err := driver.Reconnect(ctx)
	if err == nil {
		t.Error("expected error when not initialized")
	}
}

func TestFinsTCPDriver_GetDecoder(t *testing.T) {
	driver := NewFinsTCPDriver()
	decoder := driver.GetDecoder()
	if decoder != nil {
		t.Error("expected nil decoder when not initialized")
	}
}

func TestFinsTCPDriver_GetTransport(t *testing.T) {
	driver := NewFinsTCPDriver()
	transport := driver.GetTransport()
	if transport != nil {
		t.Error("expected nil transport when not initialized")
	}
}

func TestFinsTCPDriver_GetScheduler(t *testing.T) {
	driver := NewFinsTCPDriver()
	scheduler := driver.GetScheduler()
	if scheduler != nil {
		t.Error("expected nil scheduler when not initialized")
	}
}

func TestFinsTCPDriver_Disconnect_NotInitialized(t *testing.T) {
	driver := NewFinsTCPDriver()
	err := driver.Disconnect()
	if err != nil {
		t.Errorf("expected no error when disconnecting uninitialized driver, got %v", err)
	}
}

func TestFinsTCPDriver_ReadPoints_Reconnect(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	mockPLC.SetWord(MemoryAreaDMWord, 100, 9999)

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
			"timeout": 1000,
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()

	points := []Point{
		{ID: "D100", Address: "D100", DataType: DataTypeUINT16},
	}

	results, err := driver.ReadPoints(ctx, points)
	if err != nil {
		t.Fatalf("read points failed: %v", err)
	}

	if results["D100"].Quality != QualityGood {
		t.Errorf("expected good quality after auto-reconnect, got %s", results["D100"].Quality)
	}

	if results["D100"].Value != uint16(9999) {
		t.Errorf("expected D100=9999, got %v", results["D100"].Value)
	}
}

func TestFinsTCPDriver_WritePoint_Reconnect(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
			"timeout": 1000,
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()

	point := Point{ID: "D300", Address: "D300", DataType: DataTypeUINT16}
	err = driver.WritePoint(ctx, point, uint16(7777))
	if err != nil {
		t.Fatalf("write point failed: %v", err)
	}

	val := mockPLC.GetWord(MemoryAreaDMWord, 300)
	if val != 7777 {
		t.Errorf("expected D300=7777, got %d", val)
	}
}

func TestFinsTCPDriver_MixedDataTypes(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	mockPLC.SetWord(MemoryAreaDMWord, 100, 1000)
	mockPLC.SetWord(MemoryAreaDMWord, 101, 2000)
	mockPLC.SetWord(MemoryAreaCIOWord, 10, 0x8000)

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	points := []Point{
		{ID: "D100_UINT16", Address: "D100", DataType: DataTypeUINT16},
		{ID: "D100_INT32", Address: "D100", DataType: DataTypeINT32},
		{ID: "CIO10.15", Address: "CIO10.15", DataType: DataTypeBIT},
	}

	results, err := driver.ReadPoints(ctx, points)
	if err != nil {
		t.Fatalf("read points failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results["D100_UINT16"].Value != uint16(1000) {
		t.Errorf("expected D100_UINT16=1000, got %v", results["D100_UINT16"].Value)
	}

	if results["D100_INT32"].Value != int32(1000*65536+2000) {
		t.Errorf("expected D100_INT32=%d, got %v", 1000*65536+2000, results["D100_INT32"].Value)
	}

	if results["CIO10.15"].Value != true {
		t.Errorf("expected CIO10.15=true, got %v", results["CIO10.15"].Value)
	}
}

func TestFinsTCPDriver_EmptyPoints(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	results, err := driver.ReadPoints(ctx, []Point{})
	if err != nil {
		t.Fatalf("read empty points failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFinsTCPDriver_InvalidAddress(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	points := []Point{
		{ID: "invalid", Address: "INVALID123", DataType: DataTypeUINT16},
		{ID: "valid", Address: "D100", DataType: DataTypeUINT16},
	}

	results, err := driver.ReadPoints(ctx, points)
	if err != nil {
		t.Fatalf("read points failed: %v", err)
	}

	if results["invalid"].Quality != QualityBad {
		t.Errorf("expected bad quality for invalid address, got %s", results["invalid"].Quality)
	}

	if results["valid"].Quality != QualityGood {
		t.Errorf("expected good quality for valid address, got %s", results["valid"].Quality)
	}
}

func TestFinsTCPDriver_Stats(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	points := []Point{
		{ID: "D100", Address: "D100", DataType: DataTypeUINT16},
	}

	for i := 0; i < 5; i++ {
		_, err := driver.ReadPoints(ctx, points)
		if err != nil {
			t.Fatalf("read points failed: %v", err)
		}
	}

	stats := driver.GetSchedulerStats()
	if stats.TotalRequests < 5 {
		t.Errorf("expected at least 5 total requests, got %d", stats.TotalRequests)
	}
	if stats.SuccessCount < 5 {
		t.Errorf("expected at least 5 success, got %d", stats.SuccessCount)
	}
}

func TestFinsTCPDriver_StringDataType(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	writePoint := Point{ID: "D100_STR", Address: "D100.10L", DataType: DataTypeSTRING}
	if err := driver.WritePoint(ctx, writePoint, "hello"); err != nil {
		t.Fatalf("write string failed: %v", err)
	}

	readPoint := Point{ID: "D100_STR", Address: "D100.10L", DataType: DataTypeSTRING}
	results, err := driver.ReadPoints(ctx, []Point{readPoint})
	if err != nil {
		t.Fatalf("read string failed: %v", err)
	}

	if results["D100_STR"].Value != "hello" {
		t.Errorf("expected 'hello', got '%v'", results["D100_STR"].Value)
	}
}

func TestFinsTCPDriver_LongRunning(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	mockPLC.SetWord(MemoryAreaDMWord, 100, 0)

	for i := 0; i < 100; i++ {
		points := []Point{
			{ID: "D100", Address: "D100", DataType: DataTypeUINT16},
		}
		results, err := driver.ReadPoints(ctx, points)
		if err != nil {
			t.Fatalf("iteration %d: read failed: %v", i, err)
		}
		if results["D100"].Quality != QualityGood {
			t.Errorf("iteration %d: expected good quality, got %s", i, results["D100"].Quality)
		}
	}

	stats := driver.GetSchedulerStats()
	if stats.TotalRequests != 100 {
		t.Errorf("expected 100 total requests, got %d", stats.TotalRequests)
	}
}

func extractPort(addr string) int {
	_, port, err := netSplitHostPort(addr)
	if err != nil {
		return 9600
	}
	return port
}

func TestFinsTCPDriver_GetConnectionMetrics_Connected(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	metrics := driver.GetConnectionMetrics()
	if !metrics.Connected {
		t.Error("expected connected metrics")
	}
}

func TestFinsTCPDriver_GetDecoder_Initialized(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	decoder := driver.GetDecoder()
	if decoder == nil {
		t.Error("expected non-nil decoder when initialized")
	}
}

func TestFinsTCPDriver_GetTransport_Initialized(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	transport := driver.GetTransport()
	if transport == nil {
		t.Error("expected non-nil transport when initialized")
	}
}

func TestFinsTCPDriver_GetScheduler_Initialized(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	scheduler := driver.GetScheduler()
	if scheduler == nil {
		t.Error("expected non-nil scheduler when initialized")
	}
}

func TestFinsTCPDriver_WritePoint_InvalidAddress(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	driver := NewFinsTCPDriver()
	cfg := DriverConfig{
		Config: map[string]interface{}{
			"plcIP":   "127.0.0.1",
			"plcPort": extractPort(addr),
		},
	}

	if err := driver.Init(cfg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ctx := context.Background()
	if err := driver.Connect(ctx); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer driver.Disconnect()

	point := Point{ID: "invalid", Address: "INVALID", DataType: DataTypeUINT16}
	err = driver.WritePoint(ctx, point, uint16(100))
	if err == nil {
		t.Error("expected error for invalid address")
	}
}
