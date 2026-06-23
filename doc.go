// Package fins provides a comprehensive Omron FINS TCP protocol implementation
// for communicating with Omron PLCs.
//
// # Overview
//
// This package implements the Omron FINS (Factory Interface Network Service)
// TCP protocol client, allowing Go applications to read from and write to
// Omron PLC memory areas. The implementation follows a layered architecture
// with clear separation of concerns:
//
//   - Protocol Layer: FINS TCP frame encoding/decoding, protocol constants
//   - Transport Layer: TCP connection management, frame transmission, heartbeat
//   - Decoder Layer: Address parsing, data type encoding/decoding
//   - Scheduler Layer: Batch point optimization, request grouping, statistics
//   - Driver Layer: Standard driver interface for external system integration
//
// # Supported Memory Areas
//
// The driver supports the following Omron PLC memory areas:
//
//   - CIO: Input/Output relays (read/write, bit/word)
//   - W: Work relays (read/write, bit/word)
//   - H: Holding relays (read/write, bit/word)
//   - A: Auxiliary relays (read-only, bit/word)
//   - D: Data Memory (read/write, bit/word, string)
//   - P: PVs (read/write, bit/word)
//   - F: Flag area (read-only, bit)
//   - EM0-EM15: Extended Memory (read/write, bit/word, string)
//   - T/C: Timer/Counter (read/write)
//
// # Supported Data Types
//
//   - BIT: Single bit access
//   - UINT8/INT8: 8-bit integers
//   - UINT16/INT16: 16-bit integers
//   - UINT32/INT32: 32-bit integers
//   - UINT64/INT64: 64-bit integers
//   - FLOAT/DOUBLE: Floating point numbers
//   - STRING: String data (with configurable byte order)
//
// # Address Format
//
// Address strings follow the format: AREA ADDRESS[.BIT][.LEN[H][L]]
//
// Examples:
//
//	D100          - Data Memory area, word address 100
//	CIO0.0        - CIO area, address 0, bit 0
//	D10.20        - D area, address 10, string length 20 (default byte order L)
//	D10.20H       - D area, address 10, string length 20, byte order H (high first)
//	EM10W100      - EM10 area, word address 100
//	EM10W100.5    - EM10 area, address 100, bit 5
//	F0            - Flag area, bit address 0 (uint8 access)
//
// Quick Start
//
//	package main
//
//	import (
//	    "context"
//	    "fmt"
//	    "github.com/anviod/fins"
//	)
//
//	func main() {
//	    driver := fins.NewFinsTCPDriver()
//
//	    cfg := fins.DriverConfig{
//	        Protocol: "omron-fins-tcp",
//	        Config: map[string]interface{}{
//	            "plcIP":   "192.168.1.100",
//	            "plcPort": 9600,
//	            "timeout": 3000,
//	        },
//	    }
//
//	    if err := driver.Init(cfg); err != nil {
//	        panic(err)
//	    }
//
//	    ctx := context.Background()
//	    if err := driver.Connect(ctx); err != nil {
//	        panic(err)
//	    }
//	    defer driver.Disconnect()
//
//	    points := []fins.Point{
//	        {ID: "temp", Address: "D100", DataType: fins.DataTypeFLOAT, Attribute: "Read"},
//	        {ID: "status", Address: "CIO0.0", DataType: fins.DataTypeBIT, Attribute: "Read"},
//	    }
//
//	    results, err := driver.ReadPoints(ctx, points)
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    for id, val := range results {
//	        fmt.Printf("%s: %v (quality: %s)\n", id, val.Value, val.Quality)
//	    }
//	}
//
// # Connection Management
//
// The driver includes automatic reconnection with exponential backoff.
// If the connection is lost during operation, the driver will attempt
// to reconnect automatically on the next operation.
//
// Heartbeat mechanism monitors connection health periodically. The
// heartbeat interval can be configured via the heartbeatInterval
// configuration parameter (in milliseconds). Set to 0 to disable.
//
// # Error Handling
//
// All operations return descriptive errors. FINS protocol errors include
// both the error code and human-readable description. The driver implements
// 分级错误处理策略 (hierarchical error handling):
//
//   - Connection errors: Automatic reconnection with exponential backoff
//   - Protocol errors: Returned directly to caller for appropriate handling
//   - Data errors: Marked with Bad quality in point values
//   - Timeout errors: Configurable timeout with retry capability
//
// # Thread Safety
//
// The driver is designed to be safe for concurrent use. The transport
// layer uses mutex protection for write operations, and the scheduler
// uses atomic counters for statistics. However, it is recommended to
// use separate driver instances for highly concurrent scenarios.
//
// # Performance Optimization
//
// The scheduler layer automatically groups points by memory area and
// address to minimize the number of TCP requests. Points in contiguous
// memory locations are read in a single batch request, significantly
// improving throughput for large numbers of points.
//
// # Configuration Parameters
//
//   - plcIP: PLC IPv4 address (required)
//   - plcPort: PLC port number (default: 9600)
//   - timeout: Communication timeout in milliseconds (default: 3000)
//   - maxFrameLength: Maximum words per read request (default: 64)
//   - heartbeatInterval: Heartbeat interval in ms, 0 to disable (default: 30000)
//   - maxRetries: Maximum reconnection attempts (default: 3)
//   - retryInterval: Base retry interval in ms (default: 1000)
//   - srcNetworkAddr: Source network address (default: 0)
//   - srcNodeAddr: Source node address (default: 1)
//   - srcUnitAddr: Source unit address (default: 255)
//   - dstNetworkAddr: Destination network address (default: 0)
//   - dstNodeAddr: Destination node address (default: 1)
//   - dstUnitAddr: Destination unit address (default: 0)
//
// # Core Types
//
// ## Driver Interface
//
// The Driver interface defines the standard API for PLC communication:
//
//	type Driver interface {
//	    Init(cfg DriverConfig) error
//	    Connect(ctx context.Context) error
//	    Disconnect() error
//	    ReadPoints(ctx context.Context, points []Point) (map[string]Value, error)
//	    WritePoint(ctx context.Context, point Point, value interface{}) error
//	    Health() HealthStatus
//	    SetSlaveID(slaveID uint8) error
//	    SetDeviceConfig(config map[string]interface{}) error
//	    GetConnectionMetrics() ConnectionMetrics
//	}
//
// ## Point Structure
//
//	Point represents a single data point in the PLC:
//
//	type Point struct {
//	    ID       string      // Unique identifier for the point
//	    Address  string      // PLC address (e.g., "D100", "CIO0.0")
//	    DataType DataType    // Data type (e.g., DataTypeUINT16, DataTypeFLOAT)
//	    Attribute string     // Read/Write attribute
//	}
//
// ## Value Structure
//
//	Value represents the result of reading a point:
//
//	type Value struct {
//	    Value   interface{}  // The actual value read from PLC
//	    Quality Quality      // Quality status (QualityGood, QualityBad)
//	    TS      time.Time    // Timestamp of the read operation
//	}
//
// ## Connection Metrics
//
//	ConnectionMetrics provides connection status information:
//
//	type ConnectionMetrics struct {
//	    Connected        bool      // Whether the connection is established
//	    RemoteAddr       string    // Remote PLC address
//	    ConnectTime      time.Time // Time when connection was established
//	    LastDisconnectTime time.Time // Time when last disconnection occurred
//	    TotalReconnects  int       // Total number of reconnection attempts
//	}
//
// # Write Operation Example
//
//	package main
//
//	import (
//	    "context"
//	    "github.com/anviod/fins"
//	)
//
//	func main() {
//	    driver := fins.NewFinsTCPDriver()
//	    // ... init and connect ...
//
//	    ctx := context.Background()
//
//	    // Write a UINT16 value
//	    point := fins.Point{
//	        ID: "output", Address: "D200", DataType: fins.DataTypeUINT16,
//	    }
//	    err := driver.WritePoint(ctx, point, uint16(1234))
//
//	    // Write a float value
//	    point2 := fins.Point{
//	        ID: "setpoint", Address: "D300", DataType: fins.DataTypeFLOAT,
//	    }
//	    err = driver.WritePoint(ctx, point2, float32(25.5))
//
//	    // Write a bit value
//	    point3 := fins.Point{
//	        ID: "coil", Address: "CIO100.0", DataType: fins.DataTypeBIT,
//	    }
//	    err = driver.WritePoint(ctx, point3, true)
//	}
//
// # Health Monitoring Example
//
//	package main
//
//	import (
//	    "fmt"
//	    "time"
//	    "github.com/anviod/fins"
//	)
//
//	func monitorHealth(driver *fins.FinsTCPDriver) {
//	    ticker := time.NewTicker(10 * time.Second)
//	    defer ticker.Stop()
//
//	    for range ticker.C {
//	        status := driver.Health()
//	        metrics := driver.GetConnectionMetrics()
//	        fmt.Printf("Health: %s, Connected: %v, Reconnects: %d\n",
//	            status, metrics.Connected, metrics.TotalReconnects)
//	    }
//	}
//
// # Advanced Configuration Example
//
//	cfg := fins.DriverConfig{
//	    Config: map[string]interface{}{
//	        "plcIP":             "192.168.1.100",
//	        "plcPort":           9600,
//	        "timeout":           5000,
//	        "heartbeatInterval": 15000,
//	        "maxFrameLength":    128,
//	        "maxRetries":        5,
//	        "retryInterval":     2000,
//	        "srcNodeAddr":       2,
//	        "dstNodeAddr":       10,
//	    },
//	}
//
// # Testing
//
// The package includes comprehensive unit tests for each module.
// A mock PLC server is provided for integration testing without
// requiring physical hardware. Run tests with:
//
//	go test -v -cover
//
// To run specific module tests:
//
//	go test -v -run TestTransport
//	go test -v -run TestDecoder
//	go test -v -run TestScheduler
//
// # Error Codes
//
// FINS protocol error codes are defined in protocol.go. Use EndCodeText()
// to get human-readable error descriptions:
//
//	code := END_CODE_ADDR_RANGE_EXCEED
//	fmt.Println(fins.EndCodeText(code)) // "address range exceeded"
//
// # Debugging
//
// For debugging purposes, enable verbose logging by setting the
// FINS_DEBUG environment variable:
//
//	export FINS_DEBUG=1
//	go run your_program.go
//
// # See Also
//
//   - Omron FINS Command Reference: W342-E1-15
//   - CP Series Communication: W343
//   - NJ/NX Series FINS: W527
package fins