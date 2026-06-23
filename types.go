package fins

import (
	"context"
	"time"
)

type DataType string

const (
	DataTypeBIT    DataType = "BIT"
	DataTypeUINT8  DataType = "UINT8"
	DataTypeINT8   DataType = "INT8"
	DataTypeUINT16 DataType = "UINT16"
	DataTypeINT16  DataType = "INT16"
	DataTypeUINT32 DataType = "UINT32"
	DataTypeINT32  DataType = "INT32"
	DataTypeUINT64 DataType = "UINT64"
	DataTypeINT64  DataType = "INT64"
	DataTypeFLOAT  DataType = "FLOAT"
	DataTypeDOUBLE DataType = "DOUBLE"
	DataTypeSTRING DataType = "STRING"
)

type Quality string

const (
	QualityGood      Quality = "Good"
	QualityBad       Quality = "Bad"
	QualityUncertain Quality = "Uncertain"
)

type Value struct {
	Value   interface{}
	Quality Quality
	TS      time.Time
}

type Point struct {
	ID        string
	Address   string
	DataType  DataType
	Attribute string
}

type DriverConfig struct {
	Protocol string
	Config   map[string]interface{}
}

type HealthStatus string

const (
	HealthStatusUp      HealthStatus = "UP"
	HealthStatusDown    HealthStatus = "DOWN"
	HealthStatusUnknown HealthStatus = "UNKNOWN"
)

type ConnectionMetrics struct {
	Connected          bool
	ConnectTime        time.Time
	LastDisconnectTime time.Time
	ReconnectCount     int32
	LocalAddr          string
	RemoteAddr         string
}

type Driver interface {
	Init(cfg DriverConfig) error
	Connect(ctx context.Context) error
	Disconnect() error
	ReadPoints(ctx context.Context, points []Point) (map[string]Value, error)
	WritePoint(ctx context.Context, point Point, value interface{}) error
	Health() HealthStatus
	SetSlaveID(slaveID uint8) error
	SetDeviceConfig(config map[string]interface{}) error
	GetConnectionMetrics() ConnectionMetrics
}

type SchedulerStats struct {
	TotalRequests int64
	SuccessCount  int64
	FailureCount  int64
}

type ByteOrder int

const (
	ByteOrderHigh ByteOrder = iota
	ByteOrderLow
)

type ParsedAddress struct {
	Area      string
	AreaCode  uint8
	EMNumber  int
	Address   int
	Bit       int
	StringLen int
	ByteOrder ByteOrder
	IsBit     bool
	IsString  bool
}

type AreaInfo struct {
	Code          uint8
	Name          string
	ReadOnly      bool
	SupportBit    bool
	SupportString bool
}
