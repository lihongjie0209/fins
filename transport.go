package fins

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrNotConnected     = errors.New("not connected")
	ErrConnectionClosed = errors.New("connection closed")
	ErrResponseTimeout  = errors.New("response timeout")
	ErrInvalidResponse  = errors.New("invalid response")
)

type Transport struct {
	cfg map[string]interface{}

	conn   net.Conn
	reader *bufio.Reader

	connected atomic.Bool
	closed    atomic.Bool

	connectTime        time.Time
	lastDisconnectTime time.Time
	reconnectCount     atomic.Int32

	plcIP   string
	plcPort int
	timeout time.Duration

	maxRetries    int
	retryInterval time.Duration
	maxBackoff    time.Duration

	heartbeatInterval time.Duration
	stopHeartbeat     chan struct{}
	heartbeatRunning  atomic.Bool

	srcAddr FINSAddress
	dstAddr FINSAddress

	sidCounter uint8
	sidMu      sync.Mutex

	mu sync.Mutex

	responseChans map[uint8]chan *FINSFrame
	respMu        sync.Mutex
}

func NewTransport(cfg map[string]interface{}) (*Transport, error) {
	t := &Transport{
		cfg:               cfg,
		stopHeartbeat:     make(chan struct{}, 1),
		responseChans:     make(map[uint8]chan *FINSFrame),
		maxRetries:        3,
		retryInterval:     1000 * time.Millisecond,
		maxBackoff:        30 * time.Second,
		heartbeatInterval: 30 * time.Second,
		timeout:           3000 * time.Millisecond,
		plcPort:           9600,
	}

	if err := t.loadConfig(cfg); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Transport) loadConfig(cfg map[string]interface{}) error {
	if v, ok := cfg["plcIP"].(string); ok {
		t.plcIP = v
	} else {
		return errors.New("plcIP is required")
	}

	if v, ok := cfg["plcPort"].(float64); ok {
		t.plcPort = int(v)
	} else if v, ok := cfg["plcPort"].(int); ok {
		t.plcPort = v
	}

	if v, ok := cfg["timeout"].(float64); ok {
		t.timeout = time.Duration(v) * time.Millisecond
	} else if v, ok := cfg["timeout"].(int); ok {
		t.timeout = time.Duration(v) * time.Millisecond
	}

	if v, ok := cfg["heartbeatInterval"].(float64); ok {
		t.heartbeatInterval = time.Duration(v) * time.Millisecond
	} else if v, ok := cfg["heartbeatInterval"].(int); ok {
		t.heartbeatInterval = time.Duration(v) * time.Millisecond
	}

	if v, ok := cfg["maxRetries"].(float64); ok {
		t.maxRetries = int(v)
	} else if v, ok := cfg["maxRetries"].(int); ok {
		t.maxRetries = v
	}

	if v, ok := cfg["retryInterval"].(float64); ok {
		t.retryInterval = time.Duration(v) * time.Millisecond
	} else if v, ok := cfg["retryInterval"].(int); ok {
		t.retryInterval = time.Duration(v) * time.Millisecond
	}

	srcNet := 0
	srcNode := 1
	srcUnit := 255
	dstNet := 0
	dstNode := 1
	dstUnit := 0

	if v, ok := cfg["srcNetworkAddr"].(float64); ok {
		srcNet = int(v)
	} else if v, ok := cfg["srcNetworkAddr"].(int); ok {
		srcNet = v
	}
	if v, ok := cfg["srcNodeAddr"].(float64); ok {
		srcNode = int(v)
	} else if v, ok := cfg["srcNodeAddr"].(int); ok {
		srcNode = v
	}
	if v, ok := cfg["srcUnitAddr"].(float64); ok {
		srcUnit = int(v)
	} else if v, ok := cfg["srcUnitAddr"].(int); ok {
		srcUnit = v
	}

	if v, ok := cfg["dstNetworkAddr"].(float64); ok {
		dstNet = int(v)
	} else if v, ok := cfg["dstNetworkAddr"].(int); ok {
		dstNet = v
	}
	if v, ok := cfg["dstNodeAddr"].(float64); ok {
		dstNode = int(v)
	} else if v, ok := cfg["dstNodeAddr"].(int); ok {
		dstNode = v
	}
	if v, ok := cfg["dstUnitAddr"].(float64); ok {
		dstUnit = int(v)
	} else if v, ok := cfg["dstUnitAddr"].(int); ok {
		dstUnit = v
	}

	t.srcAddr = FINSAddress{Network: uint8(srcNet), Node: uint8(srcNode), Unit: uint8(srcUnit)}
	t.dstAddr = FINSAddress{Network: uint8(dstNet), Node: uint8(dstNode), Unit: uint8(dstUnit)}

	return nil
}

func (t *Transport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected.Load() {
		return nil
	}

	t.closed.Store(false)

	addr := fmt.Sprintf("%s:%d", t.plcIP, t.plcPort)
	dialer := net.Dialer{Timeout: t.timeout}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	t.conn = conn
	t.reader = bufio.NewReader(conn)
	t.connected.Store(true)
	t.connectTime = time.Now()

	go t.readLoop()

	if t.heartbeatInterval > 0 {
		t.startHeartbeat()
	}

	return nil
}

func (t *Transport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected.Load() {
		return nil
	}

	t.closed.Store(true)
	select {
	case t.stopHeartbeat <- struct{}{}:
	default:
	}

	if t.conn != nil {
		t.conn.Close()
	}

	t.connected.Store(false)
	t.lastDisconnectTime = time.Now()

	t.respMu.Lock()
	for sid, ch := range t.responseChans {
		close(ch)
		delete(t.responseChans, sid)
	}
	t.respMu.Unlock()

	return nil
}

func (t *Transport) IsConnected() bool {
	return t.connected.Load()
}

func (t *Transport) readLoop() {
	defer func() {
		if t.connected.Load() && !t.closed.Load() {
			t.connected.Store(false)
			t.lastDisconnectTime = time.Now()
		}
	}()

	buf := make([]byte, 4096)
	accumulator := make([]byte, 0)

	for {
		if t.closed.Load() {
			return
		}

		t.conn.SetReadDeadline(time.Now().Add(t.timeout * 10))

		n, err := t.reader.Read(buf)
		if err != nil {
			if t.closed.Load() {
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		if n > 0 {
			accumulator = append(accumulator, buf[:n]...)

			for {
				if len(accumulator) < 12 {
					break
				}

				tcpFrame, err := DecodeFINSTCPFrame(accumulator)
				if err != nil {
					accumulator = accumulator[1:]
					continue
				}

				frameLen := 12 + int(tcpFrame.Length) - 4
				if len(accumulator) < frameLen {
					break
				}

				finsFrame, err := DecodeFINSFrame(tcpFrame.Data)
				if err == nil {
					t.dispatchResponse(finsFrame)
				}

				if frameLen >= len(accumulator) {
					accumulator = accumulator[:0]
					break
				}
				accumulator = accumulator[frameLen:]
			}
		}
	}
}

func (t *Transport) dispatchResponse(frame *FINSFrame) {
	t.respMu.Lock()
	ch, ok := t.responseChans[frame.Header.SID]
	if ok {
		delete(t.responseChans, frame.Header.SID)
	}
	t.respMu.Unlock()

	if ok {
		select {
		case ch <- frame:
		default:
		}
	}
}

func (t *Transport) nextSID() uint8 {
	t.sidMu.Lock()
	defer t.sidMu.Unlock()
	t.sidCounter++
	if t.sidCounter == 0 {
		t.sidCounter = 1
	}
	return t.sidCounter
}

func (t *Transport) SendCommand(ctx context.Context, commandData []byte) (*FINSFrame, error) {
	if !t.connected.Load() {
		return nil, ErrNotConnected
	}

	sid := t.nextSID()

	respCh := make(chan *FINSFrame, 1)
	t.respMu.Lock()
	t.responseChans[sid] = respCh
	t.respMu.Unlock()

	defer func() {
		t.respMu.Lock()
		delete(t.responseChans, sid)
		t.respMu.Unlock()
	}()

	header := DefaultCommandHeader(t.srcAddr, t.dstAddr, sid)
	finsFrame := EncodeFINSFrame(header, binary.BigEndian.Uint16(commandData[0:2]), 0, commandData[2:])
	tcpFrame := EncodeFINSTCPFrame(TCPCommandFINSFrame, 0, finsFrame)

	t.mu.Lock()
	if t.conn == nil {
		t.mu.Unlock()
		return nil, ErrNotConnected
	}
	t.conn.SetWriteDeadline(time.Now().Add(t.timeout))
	_, err := t.conn.Write(tcpFrame)
	t.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	timeout := t.timeout
	if deadline, ok := ctx.Deadline(); ok {
		ctxTimeout := time.Until(deadline)
		if ctxTimeout < timeout {
			timeout = ctxTimeout
		}
	}

	select {
	case resp := <-respCh:
		if resp == nil {
			return nil, ErrConnectionClosed
		}
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, ErrResponseTimeout
	}
}

func (t *Transport) ReadMemory(ctx context.Context, area uint8, address uint16, count uint16) ([]byte, error) {
	memAddr := NewMemoryAddress(area, address)
	cmd := EncodeReadCommand(memAddr, count)

	resp, err := t.SendCommand(ctx, cmd)
	if err != nil {
		return nil, err
	}

	if resp.EndCode != END_CODE_NORMAL {
		return nil, fmt.Errorf("FINS error: %s (0x%04x)", EndCodeText(resp.EndCode), resp.EndCode)
	}

	return resp.Data, nil
}

func (t *Transport) WriteMemory(ctx context.Context, area uint8, address uint16, count uint16, data []byte) error {
	memAddr := NewMemoryAddress(area, address)
	cmd := EncodeWriteCommand(memAddr, count, data)

	resp, err := t.SendCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if resp.EndCode != END_CODE_NORMAL {
		return fmt.Errorf("FINS error: %s (0x%04x)", EndCodeText(resp.EndCode), resp.EndCode)
	}

	return nil
}

func (t *Transport) ReadBits(ctx context.Context, area uint8, address uint16, bitOffset uint8, count uint16) ([]bool, error) {
	memAddr := NewMemoryAddressWithBit(area, address, bitOffset)
	cmd := EncodeReadCommand(memAddr, count)

	resp, err := t.SendCommand(ctx, cmd)
	if err != nil {
		return nil, err
	}

	if resp.EndCode != END_CODE_NORMAL {
		return nil, fmt.Errorf("FINS error: %s (0x%04x)", EndCodeText(resp.EndCode), resp.EndCode)
	}

	bits := make([]bool, count)
	for i := 0; i < int(count) && i < len(resp.Data); i++ {
		bits[i] = resp.Data[i]&0x01 != 0
	}

	return bits, nil
}

func (t *Transport) WriteBits(ctx context.Context, area uint8, address uint16, bitOffset uint8, values []bool) error {
	memAddr := NewMemoryAddressWithBit(area, address, bitOffset)
	data := make([]byte, len(values))
	for i, v := range values {
		if v {
			data[i] = 0x01
		} else {
			data[i] = 0x00
		}
	}
	cmd := EncodeWriteCommand(memAddr, uint16(len(values)), data)

	resp, err := t.SendCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if resp.EndCode != END_CODE_NORMAL {
		return fmt.Errorf("FINS error: %s (0x%04x)", EndCodeText(resp.EndCode), resp.EndCode)
	}

	return nil
}

func (t *Transport) startHeartbeat() {
	if t.heartbeatRunning.Load() {
		return
	}
	t.heartbeatRunning.Store(true)

	go func() {
		ticker := time.NewTicker(t.heartbeatInterval)
		defer ticker.Stop()
		defer t.heartbeatRunning.Store(false)

		for {
			select {
			case <-ticker.C:
				if !t.connected.Load() || t.closed.Load() {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
				_, err := t.SendCommand(ctx, []byte{0x06, 0x01})
				cancel()
				if err != nil {
				}
			case <-t.stopHeartbeat:
				return
			}
		}
	}()
}

func (t *Transport) Reconnect(ctx context.Context) error {
	if err := t.Disconnect(); err != nil {
	}

	interval := t.retryInterval
	for attempt := 0; attempt < t.maxRetries; attempt++ {
		err := t.Connect(ctx)
		if err == nil {
			t.reconnectCount.Add(1)
			return nil
		}

		if attempt < t.maxRetries-1 {
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return ctx.Err()
			}
			interval *= 2
			if interval > t.maxBackoff {
				interval = t.maxBackoff
			}
		}
	}

	return fmt.Errorf("failed to reconnect after %d attempts", t.maxRetries)
}

func (t *Transport) GetMetrics() ConnectionMetrics {
	var localAddr, remoteAddr string
	if t.conn != nil {
		if t.conn.LocalAddr() != nil {
			localAddr = t.conn.LocalAddr().String()
		}
		if t.conn.RemoteAddr() != nil {
			remoteAddr = t.conn.RemoteAddr().String()
		}
	}

	return ConnectionMetrics{
		Connected:          t.connected.Load(),
		ConnectTime:        t.connectTime,
		LastDisconnectTime: t.lastDisconnectTime,
		ReconnectCount:     t.reconnectCount.Load(),
		LocalAddr:          localAddr,
		RemoteAddr:         remoteAddr,
	}
}

func (t *Transport) SetTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timeout = timeout
}

func (t *Transport) GetTimeout() time.Duration {
	return t.timeout
}

func (t *Transport) SrcAddress() FINSAddress {
	return t.srcAddr
}

func (t *Transport) DstAddress() FINSAddress {
	return t.dstAddr
}

func readFull(r io.Reader, buf []byte) error {
	_, err := io.ReadFull(r, buf)
	return err
}
