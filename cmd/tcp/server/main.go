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
)

const (
	MockDMWordSize  = 32768
	MockCIOWordSize = 6144
	MockWWordSize   = 512
	MockHWordSize   = 512
	MockAWordSize   = 512
)

type FINSAddress struct {
	Network byte
	Node    byte
	Unit    byte
}

type FINSFrame struct {
	Header  FINSHeader
	Command uint16
	EndCode uint16
	Data    []byte
}

type FINSHeader struct {
	ICF       byte
	Rsv       byte
	Gateway   byte
	Dst       FINSAddress
	Src       FINSAddress
	ServiceID byte
}

type PLCServer struct {
	listener net.Listener
	closed   bool
	mu       sync.Mutex

	dmArea  []byte
	cioArea []byte
	wArea   []byte
	hArea   []byte
	aArea   []byte

	srcAddr FINSAddress
	dstAddr FINSAddress
}

func NewPLCServer(addr string) (*PLCServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &PLCServer{
		listener: listener,
		dmArea:   make([]byte, MockDMWordSize*2),
		cioArea:  make([]byte, MockCIOWordSize*2),
		wArea:    make([]byte, MockWWordSize*2),
		hArea:    make([]byte, MockHWordSize*2),
		aArea:    make([]byte, MockAWordSize*2),
		srcAddr:  FINSAddress{Network: 0, Node: 1, Unit: 0},
		dstAddr:  FINSAddress{Network: 0, Node: 2, Unit: 0},
	}

	return s, nil
}

func (s *PLCServer) Start() {
	fmt.Printf("=== Omron FINS TCP Server ===\n")
	fmt.Printf("Listening on %s\n", s.listener.Addr().String())
	fmt.Println("Press Ctrl+C to stop")

	go s.acceptLoop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	s.Close()
	fmt.Println("\nServer stopped")
}

func (s *PLCServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed {
				return
			}
			log.Printf("Accept error: %v", err)
			continue
		}
		fmt.Printf("New connection from %s\n", conn.RemoteAddr())
		go s.handleConnection(conn)
	}
}

func (s *PLCServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("Connection closed from %s\n", conn.RemoteAddr())

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

			if string(accumulator[0:4]) != "FINS" {
				accumulator = accumulator[1:]
				continue
			}

			frameLen := int(binary.BigEndian.Uint32(accumulator[4:8]))
			totalLen := 12 + frameLen - 4

			if len(accumulator) < totalLen {
				break
			}

			finsData := accumulator[12 : 12+frameLen-4]
			accumulator = accumulator[totalLen:]

			req := decodeFINSFrame(finsData)
			if req == nil {
				continue
			}

			resp := s.handleRequest(req)
			respFrame := encodeFINSFrame(resp)

			tcpFrame := encodeFINSTCPFrame(respFrame)

			_, err = conn.Write(tcpFrame)
			if err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		}
	}
}

func encodeFINSTCPFrame(finsData []byte) []byte {
	length := uint32(len(finsData) + 4)
	buf := make([]byte, 12+len(finsData))

	copy(buf[0:4], []byte("FINS"))
	binary.BigEndian.PutUint32(buf[4:8], length)
	binary.BigEndian.PutUint16(buf[8:10], 0x0000)
	binary.BigEndian.PutUint16(buf[10:12], 0x0000)
	if len(finsData) > 0 {
		copy(buf[12:], finsData)
	}

	return buf
}

func (s *PLCServer) handleRequest(req *FINSFrame) *FINSFrame {
	var endCode uint16
	var data []byte

	switch req.Command {
	case 0x0101:
		if len(req.Data) < 6 {
			endCode = 0x1002
			break
		}
		memAddr := decodeMemoryAddress(req.Data[0:4])
		count := binary.BigEndian.Uint16(req.Data[4:6])
		data, endCode = s.readMemory(memAddr, count)

	case 0x0102:
		if len(req.Data) < 6 {
			endCode = 0x1002
			break
		}
		memAddr := decodeMemoryAddress(req.Data[0:4])
		count := binary.BigEndian.Uint16(req.Data[4:6])
		endCode = s.writeMemory(memAddr, count, req.Data[6:])

	default:
		log.Printf("Unsupported command: 0x%04x", req.Command)
		endCode = 0x0401
	}

	return &FINSFrame{
		Header: FINSHeader{
			ICF:       req.Header.ICF | 0x40,
			Rsv:       req.Header.Rsv,
			Gateway:   req.Header.Gateway,
			Dst:       req.Header.Src,
			Src:       req.Header.Dst,
			ServiceID: req.Header.ServiceID,
		},
		Command: req.Command,
		EndCode: endCode,
		Data:    data,
	}
}

func (s *PLCServer) readMemory(memAddr memoryAddress, count uint16) ([]byte, uint16) {
	var area []byte
	var maxSize int

	switch memAddr.AreaCode {
	case 0x30:
		area = s.cioArea
		maxSize = MockCIOWordSize * 2
	case 0x31:
		area = s.wArea
		maxSize = MockWWordSize * 2
	case 0x32:
		area = s.hArea
		maxSize = MockHWordSize * 2
	case 0x33:
		area = s.aArea
		maxSize = MockAWordSize * 2
	case 0x02:
		area = s.dmArea
		maxSize = MockDMWordSize * 2
	case 0x82:
		area = s.dmArea
		maxSize = MockDMWordSize * 2
	case 0xb0:
		area = s.cioArea
		maxSize = MockCIOWordSize * 2
	case 0xb1:
		area = s.wArea
		maxSize = MockWWordSize * 2
	case 0xb2:
		area = s.hArea
		maxSize = MockHWordSize * 2
	case 0xb3:
		area = s.aArea
		maxSize = MockAWordSize * 2
	default:
		return nil, 0x1101
	}

	byteOffset := int(memAddr.Address) * 2
	byteCount := int(count) * 2

	if byteOffset+byteCount > maxSize {
		return nil, 0x1104
	}

	return area[byteOffset : byteOffset+byteCount], 0x0000
}

func (s *PLCServer) writeMemory(memAddr memoryAddress, count uint16, data []byte) uint16 {
	var area []byte
	var maxSize int

	switch memAddr.AreaCode {
	case 0x30:
		area = s.cioArea
		maxSize = MockCIOWordSize * 2
	case 0x31:
		area = s.wArea
		maxSize = MockWWordSize * 2
	case 0x32:
		area = s.hArea
		maxSize = MockHWordSize * 2
	case 0x33:
		return 0x2101
	case 0x02:
		area = s.dmArea
		maxSize = MockDMWordSize * 2
	case 0x82:
		area = s.dmArea
		maxSize = MockDMWordSize * 2
	case 0xb0:
		area = s.cioArea
		maxSize = MockCIOWordSize * 2
	case 0xb1:
		area = s.wArea
		maxSize = MockWWordSize * 2
	case 0xb2:
		area = s.hArea
		maxSize = MockHWordSize * 2
	case 0xb3:
		return 0x2101
	default:
		return 0x1101
	}

	byteOffset := int(memAddr.Address) * 2
	byteCount := int(count) * 2

	if byteOffset+byteCount > maxSize {
		return 0x1104
	}

	s.mu.Lock()
	copy(area[byteOffset:byteOffset+byteCount], data)
	s.mu.Unlock()

	return 0x0000
}

func (s *PLCServer) Close() {
	s.closed = true
	if s.listener != nil {
		s.listener.Close()
	}
}

func decodeFINSFrame(data []byte) *FINSFrame {
	if len(data) < 14 {
		return nil
	}

	return &FINSFrame{
		Header: FINSHeader{
			ICF:       data[0],
			Rsv:       data[1],
			Gateway:   data[2],
			Dst: FINSAddress{
				Network: data[3],
				Node:    data[4],
				Unit:    data[5],
			},
			Src: FINSAddress{
				Network: data[6],
				Node:    data[7],
				Unit:    data[8],
			},
			ServiceID: data[9],
		},
		Command: binary.BigEndian.Uint16(data[10:12]),
		EndCode: binary.BigEndian.Uint16(data[12:14]),
		Data:    data[14:],
	}
}

func encodeFINSFrame(frame *FINSFrame) []byte {
	data := make([]byte, 14, 14+len(frame.Data))
	data[0] = frame.Header.ICF
	data[1] = frame.Header.Rsv
	data[2] = frame.Header.Gateway
	data[3] = frame.Header.Dst.Network
	data[4] = frame.Header.Dst.Node
	data[5] = frame.Header.Dst.Unit
	data[6] = frame.Header.Src.Network
	data[7] = frame.Header.Src.Node
	data[8] = frame.Header.Src.Unit
	data[9] = frame.Header.ServiceID
	binary.BigEndian.PutUint16(data[10:12], frame.Command)
	binary.BigEndian.PutUint16(data[12:14], frame.EndCode)
	data = append(data, frame.Data...)
	return data
}

type memoryAddress struct {
	AreaCode uint8
	Address  uint16
	BitOffset uint8
}

func decodeMemoryAddress(data []byte) memoryAddress {
	return memoryAddress{
		AreaCode:  data[0],
		Address:   binary.BigEndian.Uint16(data[1:3]),
		BitOffset: data[3],
	}
}

func main() {
	addr := flag.String("addr", "0.0.0.0:9600", "TCP listen address")
	flag.Parse()

	server, err := NewPLCServer(*addr)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	server.Start()
}