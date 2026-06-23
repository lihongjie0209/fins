package fins

import (
	"encoding/binary"
	"errors"
)

var (
	ErrInvalidFINSHeader = errors.New("invalid FINS TCP header magic")
	ErrFrameTooShort     = errors.New("frame too short")
)

const (
	FINSHeaderSize      = 4
	FINSTCPHeaderSize   = 8
	FINSTCPMinFrameSize = FINSTCPHeaderSize + 10
)

const (
	TCPCommandFINSFrame uint16 = 0x0000
)

const (
	MRC_MEMORY_AREA_READ           uint16 = 0x0101
	MRC_MEMORY_AREA_WRITE          uint16 = 0x0102
	MRC_MEMORY_AREA_FILL           uint16 = 0x0103
	MRC_MULTIPLE_MEMORY_AREA_READ  uint16 = 0x0104
	MRC_CONNECTION_DATA_READ       uint16 = 0x0502
	MRC_CPU_UNIT_STATUS_READ       uint16 = 0x0601
	MRC_CLOCK_READ                 uint16 = 0x0701
	MRC_CLOCK_WRITE                uint16 = 0x0702
	MRC_ERROR_CLEAR                uint16 = 0x2101
	MRC_ERROR_LOG_READ             uint16 = 0x2102
	MRC_ERROR_LOG_CLEAR            uint16 = 0x2103
)

const (
	END_CODE_NORMAL                       uint16 = 0x0000
	END_CODE_SERVICE_INTERRUPTED          uint16 = 0x0001
	END_CODE_LOCAL_NOT_IN_NET             uint16 = 0x0101
	END_CODE_TOKEN_TIMEOUT                uint16 = 0x0102
	END_CODE_RETRIES_FAILED               uint16 = 0x0103
	END_CODE_TOO_MANY_FRAMES              uint16 = 0x0104
	END_CODE_NODE_ADDR_RANGE              uint16 = 0x0105
	END_CODE_NODE_ADDR_DUP                uint16 = 0x0106
	END_CODE_DEST_NOT_IN_NET              uint16 = 0x0201
	END_CODE_UNIT_MISSING                 uint16 = 0x0202
	END_CODE_THIRD_NODE_MISSING           uint16 = 0x0203
	END_CODE_DEST_BUSY                    uint16 = 0x0204
	END_CODE_RESPONSE_TIMEOUT             uint16 = 0x0205
	END_CODE_COMM_CTRL_ERROR              uint16 = 0x0301
	END_CODE_CPU_UNIT_ERROR               uint16 = 0x0302
	END_CODE_CONTROLLER_ERROR             uint16 = 0x0303
	END_CODE_UNIT_NUM_ERROR               uint16 = 0x0304
	END_CODE_UNDEFINED_CMD                uint16 = 0x0401
	END_CODE_NOT_SUPPORTED                uint16 = 0x0402
	END_CODE_DEST_ADDR_ERROR              uint16 = 0x0501
	END_CODE_NO_ROUTING_TABLE             uint16 = 0x0502
	END_CODE_ROUTING_TABLE_ERR            uint16 = 0x0503
	END_CODE_TOO_MANY_RELAYS              uint16 = 0x0504
	END_CODE_CMD_TOO_LONG                 uint16 = 0x1001
	END_CODE_CMD_TOO_SHORT                uint16 = 0x1002
	END_CODE_ELEMENTS_MISMATCH            uint16 = 0x1003
	END_CODE_CMD_FORMAT_ERROR             uint16 = 0x1004
	END_CODE_HEADER_ERROR                 uint16 = 0x1005
	END_CODE_AREA_MISSING                 uint16 = 0x1101
	END_CODE_ACCESS_SIZE_ERR              uint16 = 0x1102
	END_CODE_ADDR_RANGE_ERR               uint16 = 0x1103
	END_CODE_ADDR_RANGE_EXCEED            uint16 = 0x1104
	END_CODE_PROGRAM_MISSING              uint16 = 0x1106
	END_CODE_RELATIONAL_ERROR             uint16 = 0x1109
	END_CODE_DUP_DATA_ACCESS              uint16 = 0x110a
	END_CODE_RESPONSE_TOO_BIG             uint16 = 0x110b
	END_CODE_PARAMETER_ERROR              uint16 = 0x110c
	END_CODE_READ_PROTECTED               uint16 = 0x2002
	END_CODE_READ_TABLE_MISSING           uint16 = 0x2003
	END_CODE_READ_DATA_MISSING            uint16 = 0x2004
	END_CODE_READ_PROG_MISSING            uint16 = 0x2005
	END_CODE_READ_FILE_MISSING            uint16 = 0x2006
	END_CODE_READ_DATA_MISMATCH           uint16 = 0x2007
	END_CODE_WRITE_READ_ONLY              uint16 = 0x2101
	END_CODE_WRITE_PROTECTED              uint16 = 0x2102
	END_CODE_WRITE_CANT_REGISTER          uint16 = 0x2103
	END_CODE_WRITE_PROG_MISSING           uint16 = 0x2105
	END_CODE_WRITE_FILE_MISSING           uint16 = 0x2106
	END_CODE_WRITE_FILE_EXISTS            uint16 = 0x2107
	END_CODE_WRITE_CANT_CHANGE            uint16 = 0x2108
	END_CODE_NOT_EXEC_DURING_EXEC         uint16 = 0x2201
	END_CODE_NOT_EXEC_WHILE_RUNNING       uint16 = 0x2202
	END_CODE_NOT_EXEC_PROG_MODE           uint16 = 0x2203
	END_CODE_NOT_EXEC_DEBUG_MODE          uint16 = 0x2204
	END_CODE_NOT_EXEC_MONITOR_MODE        uint16 = 0x2205
	END_CODE_NOT_EXEC_RUN_MODE            uint16 = 0x2206
	END_CODE_NOT_EXEC_NOT_POLLING         uint16 = 0x2207
	END_CODE_NOT_EXEC_STEP_CANT           uint16 = 0x2208
	END_CODE_NO_SUCH_DEVICE_FILE          uint16 = 0x2301
	END_CODE_NO_SUCH_DEVICE_MEM           uint16 = 0x2302
	END_CODE_NO_SUCH_DEVICE_CLOCK         uint16 = 0x2303
	END_CODE_CANT_START_STOP_TABLE        uint16 = 0x2401
	END_CODE_UNIT_ERROR_MEMORY            uint16 = 0x2502
	END_CODE_UNIT_ERROR_IO                uint16 = 0x2503
	END_CODE_UNIT_ERROR_TOO_MANY_IO       uint16 = 0x2504
	END_CODE_UNIT_ERROR_CPU_BUS           uint16 = 0x2505
	END_CODE_UNIT_ERROR_IO_DUP            uint16 = 0x2506
	END_CODE_UNIT_ERROR_IO_BUS_ERR        uint16 = 0x2507
	END_CODE_UNIT_ERROR_SYSMAC_BUS2       uint16 = 0x2509
	END_CODE_UNIT_ERROR_CPU_BUS_UNIT      uint16 = 0x250a
	END_CODE_UNIT_ERROR_SYSMAC_BUS_NUM_DUP uint16 = 0x250d
	END_CODE_UNIT_ERROR_MEM_STATUS        uint16 = 0x250f
	END_CODE_UNIT_ERROR_SYSMAC_BUS_TERM   uint16 = 0x2510
	END_CODE_CMD_ERROR_NO_PROTECTION      uint16 = 0x2601
	END_CODE_CMD_ERROR_BAD_PASSWORD       uint16 = 0x2602
	END_CODE_CMD_ERROR_PROTECTED          uint16 = 0x2604
	END_CODE_CMD_ERROR_SERVICE_EXEC       uint16 = 0x2605
	END_CODE_CMD_ERROR_SERVICE_STOPPED    uint16 = 0x2606
	END_CODE_CMD_ERROR_NO_EXEC_RIGHT      uint16 = 0x2607
	END_CODE_CMD_ERROR_SETTINGS_NOT_COMPLETE uint16 = 0x2608
	END_CODE_CMD_ERROR_ITEMS_NOT_SET      uint16 = 0x2609
	END_CODE_CMD_ERROR_NUM_ALREADY_DEFINED uint16 = 0x260a
	END_CODE_CMD_ERROR_ERROR_WONT_CLEAR   uint16 = 0x260b
	END_CODE_ACCESS_WRITE_NO_RIGHT        uint16 = 0x3001
	END_CODE_ABORT_SERVICE_ABORTED        uint16 = 0x4001
)

const (
	MemoryAreaCIOBit                      uint8 = 0x30
	MemoryAreaARBit                       uint8 = 0x31
	MemoryAreaWRBit                       uint8 = 0x32
	MemoryAreaHRBit                       uint8 = 0x33
	MemoryAreaAuxiliaryBit                uint8 = 0x33
	MemoryAreaPVBit                       uint8 = 0x35
	MemoryAreaTimerCompletionFlag         uint8 = 0x09
	MemoryAreaTimerPV                     uint8 = 0x89
	MemoryAreaCounterCompletionFlag       uint8 = 0x09
	MemoryAreaCounterPV                   uint8 = 0x89
	MemoryAreaDMBit                       uint8 = 0x02
	MemoryAreaDMWord                      uint8 = 0x82
	MemoryAreaCIOWord                     uint8 = 0xb0
	MemoryAreaWRWord                      uint8 = 0xb1
	MemoryAreaHRWord                      uint8 = 0xb2
	MemoryAreaARWord                      uint8 = 0xb3
	MemoryAreaAuxiliaryWord               uint8 = 0xb3
	MemoryAreaPVWord                      uint8 = 0xb5
	MemoryAreaTaskBit                     uint8 = 0x06
	MemoryAreaTaskStatus                  uint8 = 0x46
	MemoryAreaFlagBit                     uint8 = 0x0b
	MemoryAreaIndexRegisterPV             uint8 = 0xdc
	MemoryAreaDataRegisterPV              uint8 = 0xbc
	MemoryAreaClockPulsesConditionFlagsBit uint8 = 0x07
	MemoryAreaEM0Word                     uint8 = 0xa0
	MemoryAreaEM1Word                     uint8 = 0xa1
	MemoryAreaEM2Word                     uint8 = 0xa2
	MemoryAreaEM3Word                     uint8 = 0xa3
	MemoryAreaEM4Word                     uint8 = 0xa4
	MemoryAreaEM5Word                     uint8 = 0xa5
	MemoryAreaEM6Word                     uint8 = 0xa6
	MemoryAreaEM7Word                     uint8 = 0xa7
	MemoryAreaEM8Word                     uint8 = 0xa8
	MemoryAreaEM9Word                     uint8 = 0xa9
	MemoryAreaEM10Word                    uint8 = 0xaa
	MemoryAreaEM11Word                    uint8 = 0xab
	MemoryAreaEM12Word                    uint8 = 0xac
	MemoryAreaEM13Word                    uint8 = 0xad
	MemoryAreaEM14Word                    uint8 = 0xae
	MemoryAreaEM15Word                    uint8 = 0xaf
	MemoryAreaEM0Bit                      uint8 = 0x20
	MemoryAreaEM1Bit                      uint8 = 0x21
	MemoryAreaEM2Bit                      uint8 = 0x22
	MemoryAreaEM3Bit                      uint8 = 0x23
	MemoryAreaEM4Bit                      uint8 = 0x24
	MemoryAreaEM5Bit                      uint8 = 0x25
	MemoryAreaEM6Bit                      uint8 = 0x26
	MemoryAreaEM7Bit                      uint8 = 0x27
	MemoryAreaEM8Bit                      uint8 = 0x28
	MemoryAreaEM9Bit                      uint8 = 0x29
	MemoryAreaEM10Bit                     uint8 = 0x2a
	MemoryAreaEM11Bit                     uint8 = 0x2b
	MemoryAreaEM12Bit                     uint8 = 0x2c
	MemoryAreaEM13Bit                     uint8 = 0x2d
	MemoryAreaEM14Bit                     uint8 = 0x2e
	MemoryAreaEM15Bit                     uint8 = 0x2f
)

const (
	icfBitBridges          uint8 = 7
	icfBitMessageType      uint8 = 6
	icfBitResponseRequired uint8 = 0
)

const (
	MessageTypeCommand  uint8 = 0
	MessageTypeResponse uint8 = 1
)

type FINSAddress struct {
	Network uint8
	Node    uint8
	Unit    uint8
}

type FINSHeader struct {
	ICF  uint8
	RSV  uint8
	GCT  uint8
	Dest FINSAddress
	Src  FINSAddress
	SID  uint8
}

type FINSTCPFrame struct {
	Magic     [4]byte
	Length    uint32
	Command   uint16
	ErrorCode uint16
	Data      []byte
}

type FINSFrame struct {
	Header      FINSHeader
	CommandCode uint16
	EndCode     uint16
	Data        []byte
}

type MemoryAddress struct {
	Area      uint8
	Address   uint16
	BitOffset uint8
}

func NewMemoryAddress(area uint8, address uint16) MemoryAddress {
	return MemoryAddress{Area: area, Address: address, BitOffset: 0}
}

func NewMemoryAddressWithBit(area uint8, address uint16, bitOffset uint8) MemoryAddress {
	return MemoryAddress{Area: area, Address: address, BitOffset: bitOffset}
}

func DecodeFINSTCPFrame(data []byte) (*FINSTCPFrame, error) {
	if len(data) < 12 {
		return nil, ErrFrameTooShort
	}

	frame := &FINSTCPFrame{}
	copy(frame.Magic[:], data[0:4])

	if frame.Magic != [4]byte{'F', 'I', 'N', 'S'} {
		return nil, ErrInvalidFINSHeader
	}

	frame.Length = binary.BigEndian.Uint32(data[4:8])
	frame.Command = binary.BigEndian.Uint16(data[8:10])
	frame.ErrorCode = binary.BigEndian.Uint16(data[10:12])

	dataLen := int(frame.Length) - 4
	if dataLen < 0 {
		dataLen = 0
	}
	if len(data) > 12 {
		remaining := len(data) - 12
		if dataLen > remaining {
			dataLen = remaining
		}
		frame.Data = make([]byte, dataLen)
		copy(frame.Data, data[12:12+dataLen])
	}

	return frame, nil
}

func EncodeFINSTCPFrame(command uint16, errorCode uint16, finsData []byte) []byte {
	length := uint32(len(finsData) + 4)
	buf := make([]byte, 12+len(finsData))

	copy(buf[0:4], []byte("FINS"))
	binary.BigEndian.PutUint32(buf[4:8], length)
	binary.BigEndian.PutUint16(buf[8:10], command)
	binary.BigEndian.PutUint16(buf[10:12], errorCode)
	if len(finsData) > 0 {
		copy(buf[12:], finsData)
	}

	return buf
}

func DecodeFINSFrame(data []byte) (*FINSFrame, error) {
	if len(data) < 14 {
		return nil, ErrFrameTooShort
	}

	frame := &FINSFrame{}
	frame.Header.ICF = data[0]
	frame.Header.RSV = data[1]
	frame.Header.GCT = data[2]
	frame.Header.Dest.Network = data[3]
	frame.Header.Dest.Node = data[4]
	frame.Header.Dest.Unit = data[5]
	frame.Header.Src.Network = data[6]
	frame.Header.Src.Node = data[7]
	frame.Header.Src.Unit = data[8]
	frame.Header.SID = data[9]

	frame.CommandCode = binary.BigEndian.Uint16(data[10:12])
	frame.EndCode = binary.BigEndian.Uint16(data[12:14])

	if len(data) > 14 {
		frame.Data = make([]byte, len(data)-14)
		copy(frame.Data, data[14:])
	}

	return frame, nil
}

func EncodeFINSFrame(header FINSHeader, commandCode uint16, endCode uint16, data []byte) []byte {
	buf := make([]byte, 14+len(data))

	buf[0] = header.ICF
	buf[1] = header.RSV
	buf[2] = header.GCT
	buf[3] = header.Dest.Network
	buf[4] = header.Dest.Node
	buf[5] = header.Dest.Unit
	buf[6] = header.Src.Network
	buf[7] = header.Src.Node
	buf[8] = header.Src.Unit
	buf[9] = header.SID

	binary.BigEndian.PutUint16(buf[10:12], commandCode)
	binary.BigEndian.PutUint16(buf[12:14], endCode)

	if len(data) > 0 {
		copy(buf[14:], data)
	}

	return buf
}

func EncodeMemoryAddress(addr MemoryAddress) []byte {
	buf := make([]byte, 4)
	buf[0] = addr.Area
	binary.BigEndian.PutUint16(buf[1:3], addr.Address)
	buf[3] = addr.BitOffset
	return buf
}

func DecodeMemoryAddress(data []byte) MemoryAddress {
	return MemoryAddress{
		Area:      data[0],
		Address:   binary.BigEndian.Uint16(data[1:3]),
		BitOffset: data[3],
	}
}

func DefaultCommandHeader(src FINSAddress, dest FINSAddress, sid uint8) FINSHeader {
	h := FINSHeader{
		RSV:  0x00,
		GCT:  0x02,
		Src:  src,
		Dest: dest,
		SID:  sid,
	}
	h.ICF = 1 << icfBitBridges
	return h
}

func DefaultResponseHeader(cmdHeader FINSHeader) FINSHeader {
	h := FINSHeader{
		RSV:  0x00,
		GCT:  cmdHeader.GCT,
		Src:  cmdHeader.Dest,
		Dest: cmdHeader.Src,
		SID:  cmdHeader.SID,
	}
	h.ICF = 1 << icfBitBridges
	h.ICF |= 1 << icfBitMessageType
	return h
}

func IsWordArea(area uint8) bool {
	switch area {
	case MemoryAreaCIOWord, MemoryAreaWRWord, MemoryAreaHRWord,
		MemoryAreaARWord, MemoryAreaDMWord, MemoryAreaPVWord,
		MemoryAreaTimerPV,
		MemoryAreaEM0Word, MemoryAreaEM1Word, MemoryAreaEM2Word,
		MemoryAreaEM3Word, MemoryAreaEM4Word, MemoryAreaEM5Word,
		MemoryAreaEM6Word, MemoryAreaEM7Word, MemoryAreaEM8Word,
		MemoryAreaEM9Word, MemoryAreaEM10Word, MemoryAreaEM11Word,
		MemoryAreaEM12Word, MemoryAreaEM13Word, MemoryAreaEM14Word,
		MemoryAreaEM15Word:
		return true
	default:
		return false
	}
}

func IsBitArea(area uint8) bool {
	switch area {
	case MemoryAreaCIOBit, MemoryAreaWRBit, MemoryAreaHRBit,
		MemoryAreaARBit, MemoryAreaDMBit, MemoryAreaPVBit,
		MemoryAreaFlagBit, MemoryAreaTaskBit,
		MemoryAreaEM0Bit, MemoryAreaEM1Bit, MemoryAreaEM2Bit,
		MemoryAreaEM3Bit, MemoryAreaEM4Bit, MemoryAreaEM5Bit,
		MemoryAreaEM6Bit, MemoryAreaEM7Bit, MemoryAreaEM8Bit,
		MemoryAreaEM9Bit, MemoryAreaEM10Bit, MemoryAreaEM11Bit,
		MemoryAreaEM12Bit, MemoryAreaEM13Bit, MemoryAreaEM14Bit,
		MemoryAreaEM15Bit:
		return true
	default:
		return false
	}
}

func EMWordArea(emNumber int) (uint8, error) {
	if emNumber < 0 || emNumber > 15 {
		return 0, errors.New("invalid EM area number, must be 0-15")
	}
	return MemoryAreaEM0Word + uint8(emNumber), nil
}

func EMBitArea(emNumber int) (uint8, error) {
	if emNumber < 0 || emNumber > 15 {
		return 0, errors.New("invalid EM area number, must be 0-15")
	}
	return MemoryAreaEM0Bit + uint8(emNumber), nil
}

func EncodeReadCommand(addr MemoryAddress, count uint16) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint16(buf[0:2], MRC_MEMORY_AREA_READ)
	copy(buf[2:6], EncodeMemoryAddress(addr))
	buf[6] = 0x00
	buf[7] = 0x00
	binary.BigEndian.PutUint16(buf[6:8], count)
	return buf
}

func EncodeWriteCommand(addr MemoryAddress, count uint16, data []byte) []byte {
	buf := make([]byte, 8+len(data))
	binary.BigEndian.PutUint16(buf[0:2], MRC_MEMORY_AREA_WRITE)
	copy(buf[2:6], EncodeMemoryAddress(addr))
	binary.BigEndian.PutUint16(buf[6:8], count)
	copy(buf[8:], data)
	return buf
}

func EndCodeText(endCode uint16) string {
	switch endCode {
	case END_CODE_NORMAL:
		return "normal completion"
	case END_CODE_SERVICE_INTERRUPTED:
		return "service interrupted"
	case END_CODE_LOCAL_NOT_IN_NET:
		return "local node not in network"
	case END_CODE_TOKEN_TIMEOUT:
		return "token timeout"
	case END_CODE_RETRIES_FAILED:
		return "retries failed"
	case END_CODE_TOO_MANY_FRAMES:
		return "too many send frames"
	case END_CODE_NODE_ADDR_RANGE:
		return "node address range error"
	case END_CODE_NODE_ADDR_DUP:
		return "node address range duplication"
	case END_CODE_DEST_NOT_IN_NET:
		return "destination node not in network"
	case END_CODE_UNIT_MISSING:
		return "unit missing"
	case END_CODE_THIRD_NODE_MISSING:
		return "third node missing"
	case END_CODE_DEST_BUSY:
		return "destination node busy"
	case END_CODE_RESPONSE_TIMEOUT:
		return "response timeout"
	case END_CODE_COMM_CTRL_ERROR:
		return "communications controller error"
	case END_CODE_CPU_UNIT_ERROR:
		return "CPU unit error"
	case END_CODE_CONTROLLER_ERROR:
		return "controller error"
	case END_CODE_UNIT_NUM_ERROR:
		return "unit number error"
	case END_CODE_UNDEFINED_CMD:
		return "undefined command"
	case END_CODE_NOT_SUPPORTED:
		return "not supported by model version"
	case END_CODE_DEST_ADDR_ERROR:
		return "destination address setting error"
	case END_CODE_NO_ROUTING_TABLE:
		return "no routing tables"
	case END_CODE_ROUTING_TABLE_ERR:
		return "routing table error"
	case END_CODE_TOO_MANY_RELAYS:
		return "too many relays"
	case END_CODE_CMD_TOO_LONG:
		return "command too long"
	case END_CODE_CMD_TOO_SHORT:
		return "command too short"
	case END_CODE_ELEMENTS_MISMATCH:
		return "elements/data don't match"
	case END_CODE_CMD_FORMAT_ERROR:
		return "command format error"
	case END_CODE_HEADER_ERROR:
		return "header error"
	case END_CODE_AREA_MISSING:
		return "area classification missing"
	case END_CODE_ACCESS_SIZE_ERR:
		return "access size error"
	case END_CODE_ADDR_RANGE_ERR:
		return "address range error"
	case END_CODE_ADDR_RANGE_EXCEED:
		return "address range exceeded"
	case END_CODE_PROGRAM_MISSING:
		return "program missing"
	case END_CODE_RELATIONAL_ERROR:
		return "relational error"
	case END_CODE_DUP_DATA_ACCESS:
		return "duplicate data access"
	case END_CODE_RESPONSE_TOO_BIG:
		return "response too big"
	case END_CODE_PARAMETER_ERROR:
		return "parameter error"
	case END_CODE_READ_PROTECTED:
		return "read not possible: protected"
	case END_CODE_READ_TABLE_MISSING:
		return "read not possible: table missing"
	case END_CODE_READ_DATA_MISSING:
		return "read not possible: data missing"
	case END_CODE_READ_PROG_MISSING:
		return "read not possible: program missing"
	case END_CODE_READ_FILE_MISSING:
		return "read not possible: file missing"
	case END_CODE_READ_DATA_MISMATCH:
		return "read not possible: data mismatch"
	case END_CODE_WRITE_READ_ONLY:
		return "write not possible: read only"
	case END_CODE_WRITE_PROTECTED:
		return "write not possible: write protected"
	case END_CODE_WRITE_CANT_REGISTER:
		return "write not possible: cannot register"
	case END_CODE_WRITE_PROG_MISSING:
		return "write not possible: program missing"
	case END_CODE_WRITE_FILE_MISSING:
		return "write not possible: file missing"
	case END_CODE_WRITE_FILE_EXISTS:
		return "write not possible: file name already exists"
	case END_CODE_WRITE_CANT_CHANGE:
		return "write not possible: cannot change"
	case END_CODE_NOT_EXEC_DURING_EXEC:
		return "not executable in current mode: during execution"
	case END_CODE_NOT_EXEC_WHILE_RUNNING:
		return "not executable in current mode: while running"
	case END_CODE_NOT_EXEC_PROG_MODE:
		return "not executable in current mode: PROGRAM mode"
	case END_CODE_NOT_EXEC_DEBUG_MODE:
		return "not executable in current mode: DEBUG mode"
	case END_CODE_NOT_EXEC_MONITOR_MODE:
		return "not executable in current mode: MONITOR mode"
	case END_CODE_NOT_EXEC_RUN_MODE:
		return "not executable in current mode: RUN mode"
	case END_CODE_NOT_EXEC_NOT_POLLING:
		return "not executable in current mode: not polling node"
	case END_CODE_NOT_EXEC_STEP_CANT:
		return "not executable in current mode: step cannot be executed"
	case END_CODE_NO_SUCH_DEVICE_FILE:
		return "no such device: file device missing"
	case END_CODE_NO_SUCH_DEVICE_MEM:
		return "no such device: memory missing"
	case END_CODE_NO_SUCH_DEVICE_CLOCK:
		return "no such device: clock missing"
	case END_CODE_CANT_START_STOP_TABLE:
		return "cannot start/stop: table missing"
	case END_CODE_UNIT_ERROR_MEMORY:
		return "unit error: memory error"
	case END_CODE_UNIT_ERROR_IO:
		return "unit error: IO error"
	case END_CODE_UNIT_ERROR_TOO_MANY_IO:
		return "unit error: too many IO points"
	case END_CODE_UNIT_ERROR_CPU_BUS:
		return "unit error: CPU bus error"
	case END_CODE_UNIT_ERROR_IO_DUP:
		return "unit error: IO duplication"
	case END_CODE_UNIT_ERROR_IO_BUS_ERR:
		return "unit error: IO bus error"
	case END_CODE_UNIT_ERROR_SYSMAC_BUS2:
		return "unit error: SYSMAC BUS/2 error"
	case END_CODE_UNIT_ERROR_CPU_BUS_UNIT:
		return "unit error: CPU bus unit error"
	case END_CODE_UNIT_ERROR_SYSMAC_BUS_NUM_DUP:
		return "unit error: SYSMAC bus number duplication"
	case END_CODE_UNIT_ERROR_MEM_STATUS:
		return "unit error: memory status error"
	case END_CODE_UNIT_ERROR_SYSMAC_BUS_TERM:
		return "unit error: SYSMAC bus terminator missing"
	case END_CODE_CMD_ERROR_NO_PROTECTION:
		return "command error: no protection"
	case END_CODE_CMD_ERROR_BAD_PASSWORD:
		return "command error: incorrect password"
	case END_CODE_CMD_ERROR_PROTECTED:
		return "command error: protected"
	case END_CODE_CMD_ERROR_SERVICE_EXEC:
		return "command error: service already executing"
	case END_CODE_CMD_ERROR_SERVICE_STOPPED:
		return "command error: service stopped"
	case END_CODE_CMD_ERROR_NO_EXEC_RIGHT:
		return "command error: no execution right"
	case END_CODE_CMD_ERROR_SETTINGS_NOT_COMPLETE:
		return "command error: settings not complete"
	case END_CODE_CMD_ERROR_ITEMS_NOT_SET:
		return "command error: necessary items not set"
	case END_CODE_CMD_ERROR_NUM_ALREADY_DEFINED:
		return "command error: number already defined"
	case END_CODE_CMD_ERROR_ERROR_WONT_CLEAR:
		return "command error: error will not clear"
	case END_CODE_ACCESS_WRITE_NO_RIGHT:
		return "access write error: no access right"
	case END_CODE_ABORT_SERVICE_ABORTED:
		return "abort: service aborted"
	default:
		return "unknown error"
	}
}
