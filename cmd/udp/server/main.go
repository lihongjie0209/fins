package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	DM_AREA_SIZE     = 32768
	CIO_AREA_SIZE    = 6144
	W_AREA_SIZE      = 512
	H_AREA_SIZE      = 512
	A_AREA_SIZE      = 512
	BIT_DM_AREA_SIZE = 32768

	CommandCodeMemoryAreaRead  = 0x0101
	CommandCodeMemoryAreaWrite = 0x0102
	CommandCodeClockRead       = 0x0701

	EndCodeNormalCompletion       = 0x0000
	EndCodeAddressRangeExceeded   = 0x0101
	EndCodeNotSupportedByModelVersion = 0x0103
	EndCodeWriteNotPossibleReadOnly = 0x0201

	MemoryAreaDMWord = 0x02
	MemoryAreaDMBit  = 0x03
	MemoryAreaCIOWord = 0x30
	MemoryAreaWRWord  = 0x31
	MemoryAreaHRWord  = 0x32
	MemoryAreaARWord  = 0x33
)

const (
	MessageTypeCommand uint8 = iota
	MessageTypeResponse
)

type finsAddress struct {
	network byte
	node    byte
	unit    byte
}

type Header struct {
	messageType      uint8
	responseRequired bool
	src              finsAddress
	dst              finsAddress
	serviceID        byte
	gatewayCount     uint8
}

type PLCServer struct {
	addr      *net.UDPAddr
	conn      *net.UDPConn
	closed    bool
	mu        sync.Mutex
	dmArea    []byte
	bitDmArea []byte
	cioArea   []byte
	wArea     []byte
	hArea     []byte
	aArea     []byte
}

func NewPLCServer(addr string) (*PLCServer, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	s := &PLCServer{
		addr:      udpAddr,
		dmArea:    make([]byte, DM_AREA_SIZE),
		bitDmArea: make([]byte, BIT_DM_AREA_SIZE),
		cioArea:   make([]byte, CIO_AREA_SIZE*2),
		wArea:     make([]byte, W_AREA_SIZE*2),
		hArea:     make([]byte, H_AREA_SIZE*2),
		aArea:     make([]byte, A_AREA_SIZE*2),
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	s.conn = conn

	return s, nil
}

func (s *PLCServer) Start() {
	fmt.Printf("=== Omron FINS UDP Server ===\n")
	fmt.Printf("Listening on %s\n", s.addr.String())
	fmt.Println("Press Ctrl+C to stop")

	go s.listenLoop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	s.Close()
	fmt.Println("\nServer stopped")
}

func (s *PLCServer) listenLoop() {
	var buf [4096]byte
	for {
		n, remote, err := s.conn.ReadFromUDP(buf[:])
		if err != nil {
			if s.closed {
				return
			}
			log.Printf("Read error: %v", err)
			continue
		}

		if n > 0 {
			req := decodeRequest(buf[:n])
			resp := s.handleRequest(req)
			_, err = s.conn.WriteToUDP(encodeResponse(resp), remote)
			if err != nil {
				log.Printf("Write error: %v", err)
			}
		}
	}
}

func (s *PLCServer) handleRequest(r request) response {
	var endCode uint16
	data := []byte{}

	switch r.commandCode {
	case CommandCodeMemoryAreaRead:
		memAddr := decodeMemoryAddress(r.data[:4])
		ic := binary.BigEndian.Uint16(r.data[4:6])

		switch memAddr.memoryArea {
		case MemoryAreaDMWord:
			if memAddr.address+ic*2 > DM_AREA_SIZE {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			data = s.dmArea[memAddr.address : memAddr.address+ic*2]
			endCode = EndCodeNormalCompletion

		case MemoryAreaDMBit:
			if memAddr.address+ic > BIT_DM_AREA_SIZE {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			start := memAddr.address + uint16(memAddr.bitOffset)
			data = s.bitDmArea[start : start+ic]
			endCode = EndCodeNormalCompletion

		case MemoryAreaCIOWord:
			if memAddr.address+ic*2 > CIO_AREA_SIZE*2 {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			data = s.cioArea[memAddr.address : memAddr.address+ic*2]
			endCode = EndCodeNormalCompletion

		case MemoryAreaWRWord:
			if memAddr.address+ic*2 > W_AREA_SIZE*2 {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			data = s.wArea[memAddr.address : memAddr.address+ic*2]
			endCode = EndCodeNormalCompletion

		case MemoryAreaHRWord:
			if memAddr.address+ic*2 > H_AREA_SIZE*2 {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			data = s.hArea[memAddr.address : memAddr.address+ic*2]
			endCode = EndCodeNormalCompletion

		case MemoryAreaARWord:
			if memAddr.address+ic*2 > A_AREA_SIZE*2 {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			data = s.aArea[memAddr.address : memAddr.address+ic*2]
			endCode = EndCodeNormalCompletion

		default:
			log.Printf("Unsupported memory area: 0x%02x", memAddr.memoryArea)
			endCode = EndCodeNotSupportedByModelVersion
		}

	case CommandCodeMemoryAreaWrite:
		memAddr := decodeMemoryAddress(r.data[:4])
		ic := binary.BigEndian.Uint16(r.data[4:6])

		switch memAddr.memoryArea {
		case MemoryAreaDMWord:
			if memAddr.address+ic*2 > DM_AREA_SIZE {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			s.mu.Lock()
			copy(s.dmArea[memAddr.address:memAddr.address+ic*2], r.data[6:6+ic*2])
			s.mu.Unlock()
			endCode = EndCodeNormalCompletion

		case MemoryAreaDMBit:
			if memAddr.address+ic > BIT_DM_AREA_SIZE {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			start := memAddr.address + uint16(memAddr.bitOffset)
			s.mu.Lock()
			copy(s.bitDmArea[start:start+ic], r.data[6:6+ic])
			s.mu.Unlock()
			endCode = EndCodeNormalCompletion

		case MemoryAreaCIOWord:
			if memAddr.address+ic*2 > CIO_AREA_SIZE*2 {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			s.mu.Lock()
			copy(s.cioArea[memAddr.address:memAddr.address+ic*2], r.data[6:6+ic*2])
			s.mu.Unlock()
			endCode = EndCodeNormalCompletion

		case MemoryAreaWRWord:
			if memAddr.address+ic*2 > W_AREA_SIZE*2 {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			s.mu.Lock()
			copy(s.wArea[memAddr.address:memAddr.address+ic*2], r.data[6:6+ic*2])
			s.mu.Unlock()
			endCode = EndCodeNormalCompletion

		case MemoryAreaHRWord:
			if memAddr.address+ic*2 > H_AREA_SIZE*2 {
				endCode = EndCodeAddressRangeExceeded
				break
			}
			s.mu.Lock()
			copy(s.hArea[memAddr.address:memAddr.address+ic*2], r.data[6:6+ic*2])
			s.mu.Unlock()
			endCode = EndCodeNormalCompletion

		case MemoryAreaARWord:
			endCode = EndCodeWriteNotPossibleReadOnly

		default:
			log.Printf("Unsupported memory area: 0x%02x", memAddr.memoryArea)
			endCode = EndCodeNotSupportedByModelVersion
		}

	case CommandCodeClockRead:
		now := time.Now()
		data = encodeBCD(uint64(now.Year() % 100))
		data = append(data, encodeBCD(uint64(now.Month()))...)
		data = append(data, encodeBCD(uint64(now.Day()))...)
		data = append(data, encodeBCD(uint64(now.Hour()))...)
		data = append(data, encodeBCD(uint64(now.Minute()))...)
		data = append(data, encodeBCD(uint64(now.Second()))...)
		endCode = EndCodeNormalCompletion

	default:
		log.Printf("Unsupported command code: 0x%04x", r.commandCode)
		endCode = EndCodeNotSupportedByModelVersion
	}

	return response{defaultResponseHeader(r.header), r.commandCode, endCode, data}
}

func (s *PLCServer) Close() {
	s.closed = true
	if s.conn != nil {
		s.conn.Close()
	}
}

func decodeRequest(bytes []byte) request {
	return request{
		decodeHeader(bytes[0:10]),
		binary.BigEndian.Uint16(bytes[10:12]),
		bytes[12:],
	}
}

func decodeResponse(bytes []byte) response {
	return response{
		decodeHeader(bytes[0:10]),
		binary.BigEndian.Uint16(bytes[10:12]),
		binary.BigEndian.Uint16(bytes[12:14]),
		bytes[14:],
	}
}

func encodeResponse(resp response) []byte {
	bytes := make([]byte, 4, 4+len(resp.data))
	binary.BigEndian.PutUint16(bytes[0:2], resp.commandCode)
	binary.BigEndian.PutUint16(bytes[2:4], resp.endCode)
	bytes = append(bytes, resp.data...)
	bh := encodeHeader(resp.header)
	bh = append(bh, bytes...)
	return bh
}

func encodeMemoryAddress(memoryAddr memoryAddress) []byte {
	bytes := make([]byte, 4, 4)
	bytes[0] = memoryAddr.memoryArea
	binary.BigEndian.PutUint16(bytes[1:3], memoryAddr.address)
	bytes[3] = memoryAddr.bitOffset
	return bytes
}

func decodeMemoryAddress(data []byte) memoryAddress {
	return memoryAddress{data[0], binary.BigEndian.Uint16(data[1:3]), data[3]}
}

const (
	icfBridgesBit          byte = 7
	icfMessageTypeBit      byte = 6
	icfResponseRequiredBit byte = 0
)

func decodeHeader(bytes []byte) Header {
	header := Header{}
	icf := bytes[0]
	if icf&1<<icfResponseRequiredBit == 0 {
		header.responseRequired = true
	}
	if icf&1<<icfMessageTypeBit == 0 {
		header.messageType = MessageTypeCommand
	} else {
		header.messageType = MessageTypeResponse
	}
	header.gatewayCount = bytes[2]
	header.dst = finsAddress{bytes[3], bytes[4], bytes[5]}
	header.src = finsAddress{bytes[6], bytes[7], bytes[8]}
	header.serviceID = bytes[9]
	return header
}

func encodeHeader(h Header) []byte {
	var icf byte
	icf = 1 << icfBridgesBit
	if !h.responseRequired {
		icf |= 1 << icfResponseRequiredBit
	}
	if h.messageType == MessageTypeResponse {
		icf |= 1 << icfMessageTypeBit
	}
	bytes := []byte{
		icf, 0x00, h.gatewayCount,
		h.dst.network, h.dst.node, h.dst.unit,
		h.src.network, h.src.node, h.src.unit,
		h.serviceID,
	}
	return bytes
}

func defaultResponseHeader(reqHeader Header) Header {
	return Header{
		responseRequired: false,
		messageType:      MessageTypeResponse,
		gatewayCount:     reqHeader.gatewayCount,
		dst:              reqHeader.src,
		src:              reqHeader.dst,
		serviceID:        reqHeader.serviceID,
	}
}

func encodeBCD(x uint64) []byte {
	if x == 0 {
		return []byte{0x00}
	}
	var n int
	for xx := x; xx > 0; n++ {
		xx = xx / 10
	}
	bcd := make([]byte, (n+1)/2)
	if n%2 == 1 {
		hi, lo := byte(x%10), byte(0x00)
		bcd[(n-1)/2] = hi<<4 | lo
		x = x / 10
		n--
	}
	for i := n/2 - 1; i >= 0; i-- {
		hi, lo := byte((x/10)%10), byte(x%10)
		bcd[i] = hi<<4 | lo
		x = x / 100
	}
	return bcd
}

func main() {
	addr := flag.String("addr", "0.0.0.0:9600", "UDP listen address")
	flag.Parse()

	server, err := NewPLCServer(*addr)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	server.Start()
}

type request struct {
	header      Header
	commandCode uint16
	data        []byte
}

type response struct {
	header      Header
	commandCode uint16
	endCode     uint16
	data        []byte
}

type memoryAddress struct {
	memoryArea byte
	address    uint16
	bitOffset  byte
}