package fins

import (
	"encoding/binary"
	"net"
	"sync"
	"sync/atomic"
)

type MockPLC struct {
	listener net.Listener
	addr     string
	closed   atomic.Bool
	mu       sync.Mutex

	dmArea  []byte
	cioArea []byte
	wArea   []byte
	hArea   []byte
	aArea   []byte

	srcAddr FINSAddress
	dstAddr FINSAddress
}

const (
	MockDMWordSize  = 32768
	MockCIOWordSize = 6144
	MockWWordSize   = 512
	MockHWordSize   = 512
	MockAWordSize   = 512
)

func NewMockPLC() *MockPLC {
	return &MockPLC{
		dmArea:  make([]byte, MockDMWordSize*2),
		cioArea: make([]byte, MockCIOWordSize*2),
		wArea:   make([]byte, MockWWordSize*2),
		hArea:   make([]byte, MockHWordSize*2),
		aArea:   make([]byte, MockAWordSize*2),
		srcAddr: FINSAddress{Network: 0, Node: 1, Unit: 0},
		dstAddr: FINSAddress{Network: 0, Node: 2, Unit: 0},
	}
}

func (m *MockPLC) Start() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	m.listener = listener
	m.addr = listener.Addr().String()

	go m.acceptLoop()

	return m.addr, nil
}

func (m *MockPLC) Addr() string {
	return m.addr
}

func (m *MockPLC) Close() {
	m.closed.Store(true)
	if m.listener != nil {
		m.listener.Close()
	}
}

func (m *MockPLC) acceptLoop() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			if m.closed.Load() {
				return
			}
			continue
		}
		go m.handleConnection(conn)
	}
}

func (m *MockPLC) handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 4096)
	accumulator := make([]byte, 0)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		if n == 0 {
			continue
		}

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
				resp := m.handleRequest(finsFrame)
				respData := EncodeFINSFrame(resp.Header, resp.CommandCode, resp.EndCode, resp.Data)
				tcpResp := EncodeFINSTCPFrame(TCPCommandFINSFrame, 0, respData)
				conn.Write(tcpResp)
			}

			if frameLen >= len(accumulator) {
				accumulator = accumulator[:0]
				break
			}
			accumulator = accumulator[frameLen:]
		}
	}
}

func (m *MockPLC) handleRequest(req *FINSFrame) *FINSFrame {
	respHeader := DefaultResponseHeader(req.Header)
	resp := &FINSFrame{
		Header:      respHeader,
		CommandCode: req.CommandCode,
		EndCode:     END_CODE_NORMAL,
	}

	switch req.CommandCode {
	case MRC_MEMORY_AREA_READ:
		if len(req.Data) < 6 {
			resp.EndCode = END_CODE_CMD_TOO_SHORT
			return resp
		}
		memAddr := DecodeMemoryAddress(req.Data[0:4])
		count := binary.BigEndian.Uint16(req.Data[4:6])
		data, endCode := m.readMemory(memAddr, count)
		resp.Data = data
		resp.EndCode = endCode

	case MRC_MEMORY_AREA_WRITE:
		if len(req.Data) < 6 {
			resp.EndCode = END_CODE_CMD_TOO_SHORT
			return resp
		}
		memAddr := DecodeMemoryAddress(req.Data[0:4])
		count := binary.BigEndian.Uint16(req.Data[4:6])
		writeData := req.Data[6:]
		endCode := m.writeMemory(memAddr, count, writeData)
		resp.EndCode = endCode

	case MRC_CPU_UNIT_STATUS_READ:
		resp.Data = []byte{0x00, 0x00}

	default:
		resp.EndCode = END_CODE_UNDEFINED_CMD
	}

	return resp
}

func (m *MockPLC) getArea(areaCode uint8) ([]byte, int) {
	switch areaCode {
	case MemoryAreaDMWord:
		return m.dmArea, MockDMWordSize
	case MemoryAreaCIOWord:
		return m.cioArea, MockCIOWordSize
	case MemoryAreaWRWord:
		return m.wArea, MockWWordSize
	case MemoryAreaHRWord:
		return m.hArea, MockHWordSize
	case MemoryAreaARWord:
		return m.aArea, MockAWordSize
	default:
		return nil, 0
	}
}

func (m *MockPLC) getBitArea(areaCode uint8) ([]byte, int) {
	switch areaCode {
	case MemoryAreaDMBit:
		return m.dmArea, MockDMWordSize * 16
	case MemoryAreaCIOBit:
		return m.cioArea, MockCIOWordSize * 16
	case MemoryAreaWRBit:
		return m.wArea, MockWWordSize * 16
	case MemoryAreaHRBit:
		return m.hArea, MockHWordSize * 16
	case MemoryAreaARBit:
		return m.aArea, MockAWordSize * 16
	default:
		return nil, 0
	}
}

func (m *MockPLC) readMemory(addr MemoryAddress, count uint16) ([]byte, uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if IsWordArea(addr.Area) {
		area, _ := m.getArea(addr.Area)
		if area == nil {
			return nil, END_CODE_NOT_SUPPORTED
		}

		startByte := int(addr.Address) * 2
		endByte := startByte + int(count)*2

		if endByte > len(area) {
			return nil, END_CODE_ADDR_RANGE_EXCEED
		}

		data := make([]byte, int(count)*2)
		copy(data, area[startByte:endByte])
		return data, END_CODE_NORMAL
	}

	if IsBitArea(addr.Area) {
		area, maxBits := m.getBitArea(addr.Area)
		if area == nil {
			return nil, END_CODE_NOT_SUPPORTED
		}

		startBit := int(addr.Address)*16 + int(addr.BitOffset)
		if startBit+int(count) > maxBits {
			return nil, END_CODE_ADDR_RANGE_EXCEED
		}

		data := make([]byte, count)
		for i := 0; i < int(count); i++ {
			bitPos := startBit + i
			wordByteIdx := bitPos / 16 * 2
			bitInWord := bitPos % 16
			bitFromLeft := 15 - bitInWord
			byteOffset := wordByteIdx + bitFromLeft/8
			bitInByte := uint(7 - bitFromLeft%8)

			if area[byteOffset]&(1<<bitInByte) != 0 {
				data[i] = 0x01
			} else {
				data[i] = 0x00
			}
		}
		return data, END_CODE_NORMAL
	}

	return nil, END_CODE_NOT_SUPPORTED
}

func (m *MockPLC) writeMemory(addr MemoryAddress, count uint16, data []byte) uint16 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if IsWordArea(addr.Area) {
		area, _ := m.getArea(addr.Area)
		if area == nil {
			return END_CODE_NOT_SUPPORTED
		}

		startByte := int(addr.Address) * 2
		endByte := startByte + int(count)*2

		if endByte > len(area) {
			return END_CODE_ADDR_RANGE_EXCEED
		}

		copy(area[startByte:endByte], data)
		return END_CODE_NORMAL
	}

	if IsBitArea(addr.Area) {
		area, maxBits := m.getBitArea(addr.Area)
		if area == nil {
			return END_CODE_NOT_SUPPORTED
		}

		startBit := int(addr.Address)*16 + int(addr.BitOffset)
		if startBit+int(count) > maxBits {
			return END_CODE_ADDR_RANGE_EXCEED
		}

		for i := 0; i < int(count) && i < len(data); i++ {
			bitPos := startBit + i
			wordByteIdx := bitPos / 16 * 2
			bitInWord := bitPos % 16
			bitFromLeft := 15 - bitInWord
			byteOffset := wordByteIdx + bitFromLeft/8
			bitInByte := uint(7 - bitFromLeft%8)

			if data[i] != 0 {
				area[byteOffset] |= 1 << bitInByte
			} else {
				area[byteOffset] &^= 1 << bitInByte
			}
		}
		return END_CODE_NORMAL
	}

	return END_CODE_NOT_SUPPORTED
}

func (m *MockPLC) SetWord(areaCode uint8, address uint16, value uint16) {
	area, _ := m.getArea(areaCode)
	if area != nil && int(address)*2+2 <= len(area) {
		m.mu.Lock()
		binary.BigEndian.PutUint16(area[address*2:address*2+2], value)
		m.mu.Unlock()
	}
}

func (m *MockPLC) GetWord(areaCode uint8, address uint16) uint16 {
	area, _ := m.getArea(areaCode)
	if area != nil && int(address)*2+2 <= len(area) {
		m.mu.Lock()
		defer m.mu.Unlock()
		return binary.BigEndian.Uint16(area[address*2 : address*2+2])
	}
	return 0
}
