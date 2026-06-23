package fins

import (
	"context"
	"errors"
	"sync"
	"time"
)

type FinsTCPDriver struct {
	config    DriverConfig
	transport *Transport
	decoder   *Decoder
	scheduler *Scheduler

	mu        sync.RWMutex
	slaveID   uint8
	deviceCfg map[string]interface{}

	initialized bool
}

func NewFinsTCPDriver() *FinsTCPDriver {
	return &FinsTCPDriver{
		deviceCfg: make(map[string]interface{}),
	}
}

func (d *FinsTCPDriver) Init(cfg DriverConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.initialized {
		return errors.New("driver already initialized")
	}

	d.config = cfg

	if cfg.Config != nil {
		for k, v := range cfg.Config {
			d.deviceCfg[k] = v
		}
	}

	decoder := NewDecoder()
	d.decoder = decoder

	transport, err := NewTransport(d.deviceCfg)
	if err != nil {
		return err
	}
	d.transport = transport

	d.scheduler = NewScheduler(transport, decoder, d.deviceCfg)

	d.initialized = true

	return nil
}

func (d *FinsTCPDriver) Connect(ctx context.Context) error {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return errors.New("driver not initialized")
	}
	d.mu.RUnlock()

	return d.transport.Connect(ctx)
}

func (d *FinsTCPDriver) Disconnect() error {
	d.mu.RLock()
	if d.transport == nil {
		d.mu.RUnlock()
		return nil
	}
	d.mu.RUnlock()

	return d.transport.Disconnect()
}

func (d *FinsTCPDriver) ReadPoints(ctx context.Context, points []Point) (map[string]Value, error) {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return nil, errors.New("driver not initialized")
	}
	if d.scheduler == nil {
		d.mu.RUnlock()
		return nil, errors.New("scheduler not initialized")
	}
	d.mu.RUnlock()

	if !d.transport.IsConnected() {
		if err := d.transport.Reconnect(ctx); err != nil {
			results := make(map[string]Value, len(points))
			now := time.Now()
			for _, p := range points {
				results[p.ID] = Value{
					Value:   nil,
					Quality: QualityBad,
					TS:      now,
				}
			}
			return results, nil
		}
	}

	return d.scheduler.ReadPoints(ctx, points)
}

func (d *FinsTCPDriver) WritePoint(ctx context.Context, point Point, value interface{}) error {
	d.mu.RLock()
	if !d.initialized {
		d.mu.RUnlock()
		return errors.New("driver not initialized")
	}
	if d.scheduler == nil {
		d.mu.RUnlock()
		return errors.New("scheduler not initialized")
	}
	d.mu.RUnlock()

	if !d.transport.IsConnected() {
		if err := d.transport.Reconnect(ctx); err != nil {
			return err
		}
	}

	return d.scheduler.WritePoint(ctx, point, value)
}

func (d *FinsTCPDriver) Health() HealthStatus {
	d.mu.RLock()
	if !d.initialized || d.transport == nil {
		d.mu.RUnlock()
		return HealthStatusUnknown
	}
	d.mu.RUnlock()

	return d.scheduler.Health()
}

func (d *FinsTCPDriver) SetSlaveID(slaveID uint8) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.slaveID = slaveID
	return nil
}

func (d *FinsTCPDriver) SetDeviceConfig(config map[string]interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for k, v := range config {
		d.deviceCfg[k] = v
	}

	return nil
}

func (d *FinsTCPDriver) GetConnectionMetrics() ConnectionMetrics {
	d.mu.RLock()
	if d.transport == nil {
		d.mu.RUnlock()
		return ConnectionMetrics{}
	}
	d.mu.RUnlock()

	return d.transport.GetMetrics()
}

func (d *FinsTCPDriver) GetSchedulerStats() SchedulerStats {
	d.mu.RLock()
	if d.scheduler == nil {
		d.mu.RUnlock()
		return SchedulerStats{}
	}
	d.mu.RUnlock()

	return d.scheduler.GetStats()
}

func (d *FinsTCPDriver) GetDecoder() *Decoder {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.decoder
}

func (d *FinsTCPDriver) GetTransport() *Transport {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.transport
}

func (d *FinsTCPDriver) GetScheduler() *Scheduler {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.scheduler
}

func (d *FinsTCPDriver) IsInitialized() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.initialized
}

func (d *FinsTCPDriver) IsConnected() bool {
	d.mu.RLock()
	if d.transport == nil {
		d.mu.RUnlock()
		return false
	}
	d.mu.RUnlock()
	return d.transport.IsConnected()
}

func (d *FinsTCPDriver) Reconnect(ctx context.Context) error {
	d.mu.RLock()
	if d.transport == nil {
		d.mu.RUnlock()
		return errors.New("transport not initialized")
	}
	d.mu.RUnlock()

	return d.transport.Reconnect(ctx)
}

var _ Driver = (*FinsTCPDriver)(nil)
