package fins

import (
	"encoding/binary"
	"testing"
)

func TestEncodeDecodeFINSTCPFrame(t *testing.T) {
	finsData := []byte{0x80, 0x00, 0x02, 0x00, 0x01, 0x00, 0x00, 0x02, 0x00, 0x01, 0x01, 0x01, 0x00, 0x00}

	encoded := EncodeFINSTCPFrame(TCPCommandFINSFrame, 0, finsData)

	if len(encoded) != 12+len(finsData) {
		t.Fatalf("expected length %d, got %d", 12+len(finsData), len(encoded))
	}

	if string(encoded[0:4]) != "FINS" {
		t.Fatalf("expected magic 'FINS', got '%s'", string(encoded[0:4]))
	}

	length := binary.BigEndian.Uint32(encoded[4:8])
	if length != uint32(len(finsData)+4) {
		t.Fatalf("expected length %d, got %d", len(finsData)+4, length)
	}

	cmd := binary.BigEndian.Uint16(encoded[8:10])
	if cmd != TCPCommandFINSFrame {
		t.Fatalf("expected command %d, got %d", TCPCommandFINSFrame, cmd)
	}

	errCode := binary.BigEndian.Uint16(encoded[10:12])
	if errCode != 0 {
		t.Fatalf("expected error code 0, got %d", errCode)
	}

	decoded, err := DecodeFINSTCPFrame(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decoded.Command != TCPCommandFINSFrame {
		t.Errorf("expected command %d, got %d", TCPCommandFINSFrame, decoded.Command)
	}

	if decoded.ErrorCode != 0 {
		t.Errorf("expected error code 0, got %d", decoded.ErrorCode)
	}

	if len(decoded.Data) != len(finsData) {
		t.Errorf("expected data length %d, got %d", len(finsData), len(decoded.Data))
	}
}

func TestDecodeFINSTCPFrame_InvalidMagic(t *testing.T) {
	badData := []byte("NOTFINS0000000000")
	_, err := DecodeFINSTCPFrame(badData)
	if err == nil {
		t.Fatal("expected error for invalid magic")
	}
}

func TestDecodeFINSTCPFrame_TooShort(t *testing.T) {
	shortData := []byte("FIN")
	_, err := DecodeFINSTCPFrame(shortData)
	if err == nil {
		t.Fatal("expected error for short frame")
	}
}

func TestEncodeDecodeFINSFrame(t *testing.T) {
	src := FINSAddress{Network: 0, Node: 2, Unit: 0}
	dst := FINSAddress{Network: 0, Node: 1, Unit: 0}
	header := DefaultCommandHeader(src, dst, 0x01)

	data := []byte{0x01, 0x02, 0x03, 0x04}
	encoded := EncodeFINSFrame(header, MRC_MEMORY_AREA_READ, END_CODE_NORMAL, data)

	if len(encoded) != 14+len(data) {
		t.Fatalf("expected length %d, got %d", 14+len(data), len(encoded))
	}

	decoded, err := DecodeFINSFrame(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decoded.CommandCode != MRC_MEMORY_AREA_READ {
		t.Errorf("expected command code %04x, got %04x", MRC_MEMORY_AREA_READ, decoded.CommandCode)
	}

	if decoded.EndCode != END_CODE_NORMAL {
		t.Errorf("expected end code %04x, got %04x", END_CODE_NORMAL, decoded.EndCode)
	}

	if decoded.Header.SID != 0x01 {
		t.Errorf("expected SID 0x01, got 0x%02x", decoded.Header.SID)
	}

	if len(decoded.Data) != len(data) {
		t.Errorf("expected data length %d, got %d", len(data), len(decoded.Data))
	}
}

func TestDecodeFINSFrame_TooShort(t *testing.T) {
	shortData := []byte{0x80, 0x00, 0x02}
	_, err := DecodeFINSFrame(shortData)
	if err == nil {
		t.Fatal("expected error for short frame")
	}
}

func TestEncodeDecodeMemoryAddress(t *testing.T) {
	addr := MemoryAddress{Area: MemoryAreaDMWord, Address: 100, BitOffset: 0}
	encoded := EncodeMemoryAddress(addr)

	if len(encoded) != 4 {
		t.Fatalf("expected length 4, got %d", len(encoded))
	}

	if encoded[0] != MemoryAreaDMWord {
		t.Errorf("expected area %02x, got %02x", MemoryAreaDMWord, encoded[0])
	}

	decoded := DecodeMemoryAddress(encoded)
	if decoded.Area != addr.Area {
		t.Errorf("expected area %02x, got %02x", addr.Area, decoded.Area)
	}
	if decoded.Address != addr.Address {
		t.Errorf("expected address %d, got %d", addr.Address, decoded.Address)
	}
	if decoded.BitOffset != addr.BitOffset {
		t.Errorf("expected bit offset %d, got %d", addr.BitOffset, decoded.BitOffset)
	}
}

func TestDefaultCommandHeader(t *testing.T) {
	src := FINSAddress{Network: 0, Node: 2, Unit: 0}
	dst := FINSAddress{Network: 0, Node: 1, Unit: 0}
	header := DefaultCommandHeader(src, dst, 0x01)

	if header.SID != 0x01 {
		t.Errorf("expected SID 0x01, got 0x%02x", header.SID)
	}

	if header.Src.Node != 2 {
		t.Errorf("expected src node 2, got %d", header.Src.Node)
	}

	if header.Dest.Node != 1 {
		t.Errorf("expected dest node 1, got %d", header.Dest.Node)
	}
}

func TestDefaultResponseHeader(t *testing.T) {
	src := FINSAddress{Network: 0, Node: 2, Unit: 0}
	dst := FINSAddress{Network: 0, Node: 1, Unit: 0}
	cmdHeader := DefaultCommandHeader(src, dst, 0x01)
	respHeader := DefaultResponseHeader(cmdHeader)

	if respHeader.SID != 0x01 {
		t.Errorf("expected SID 0x01, got 0x%02x", respHeader.SID)
	}

	if respHeader.Src.Node != 1 {
		t.Errorf("expected src node 1, got %d", respHeader.Src.Node)
	}

	if respHeader.Dest.Node != 2 {
		t.Errorf("expected dest node 2, got %d", respHeader.Dest.Node)
	}
}

func TestIsWordArea(t *testing.T) {
	if !IsWordArea(MemoryAreaDMWord) {
		t.Error("DM word should be word area")
	}
	if !IsWordArea(MemoryAreaCIOWord) {
		t.Error("CIO word should be word area")
	}
	if IsWordArea(MemoryAreaDMBit) {
		t.Error("DM bit should not be word area")
	}
}

func TestIsBitArea(t *testing.T) {
	if !IsBitArea(MemoryAreaDMBit) {
		t.Error("DM bit should be bit area")
	}
	if !IsBitArea(MemoryAreaCIOBit) {
		t.Error("CIO bit should be bit area")
	}
	if IsBitArea(MemoryAreaDMWord) {
		t.Error("DM word should not be bit area")
	}
}

func TestEMWordArea(t *testing.T) {
	code, err := EMWordArea(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != MemoryAreaEM10Word {
		t.Errorf("expected %02x, got %02x", MemoryAreaEM10Word, code)
	}

	_, err = EMWordArea(16)
	if err == nil {
		t.Error("expected error for invalid EM number")
	}

	_, err = EMWordArea(-1)
	if err == nil {
		t.Error("expected error for negative EM number")
	}
}

func TestEMBitArea(t *testing.T) {
	code, err := EMBitArea(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != MemoryAreaEM5Bit {
		t.Errorf("expected %02x, got %02x", MemoryAreaEM5Bit, code)
	}

	_, err = EMBitArea(20)
	if err == nil {
		t.Error("expected error for invalid EM number")
	}
}

func TestEndCodeText(t *testing.T) {
	tests := []struct {
		code uint16
		want string
	}{
		{END_CODE_NORMAL, "normal completion"},
		{END_CODE_SERVICE_INTERRUPTED, "service interrupted"},
		{END_CODE_LOCAL_NOT_IN_NET, "local node not in network"},
		{END_CODE_TOKEN_TIMEOUT, "token timeout"},
		{END_CODE_RETRIES_FAILED, "retries failed"},
		{END_CODE_TOO_MANY_FRAMES, "too many send frames"},
		{END_CODE_NODE_ADDR_RANGE, "node address range error"},
		{END_CODE_NODE_ADDR_DUP, "node address range duplication"},
		{END_CODE_DEST_NOT_IN_NET, "destination node not in network"},
		{END_CODE_UNIT_MISSING, "unit missing"},
		{END_CODE_THIRD_NODE_MISSING, "third node missing"},
		{END_CODE_DEST_BUSY, "destination node busy"},
		{END_CODE_RESPONSE_TIMEOUT, "response timeout"},
		{END_CODE_COMM_CTRL_ERROR, "communications controller error"},
		{END_CODE_CPU_UNIT_ERROR, "CPU unit error"},
		{END_CODE_CONTROLLER_ERROR, "controller error"},
		{END_CODE_UNIT_NUM_ERROR, "unit number error"},
		{END_CODE_UNDEFINED_CMD, "undefined command"},
		{END_CODE_NOT_SUPPORTED, "not supported by model version"},
		{END_CODE_DEST_ADDR_ERROR, "destination address setting error"},
		{END_CODE_NO_ROUTING_TABLE, "no routing tables"},
		{END_CODE_ROUTING_TABLE_ERR, "routing table error"},
		{END_CODE_TOO_MANY_RELAYS, "too many relays"},
		{END_CODE_CMD_TOO_LONG, "command too long"},
		{END_CODE_CMD_TOO_SHORT, "command too short"},
		{END_CODE_ELEMENTS_MISMATCH, "elements/data don't match"},
		{END_CODE_CMD_FORMAT_ERROR, "command format error"},
		{END_CODE_HEADER_ERROR, "header error"},
		{END_CODE_AREA_MISSING, "area classification missing"},
		{END_CODE_ACCESS_SIZE_ERR, "access size error"},
		{END_CODE_ADDR_RANGE_ERR, "address range error"},
		{END_CODE_ADDR_RANGE_EXCEED, "address range exceeded"},
		{END_CODE_PROGRAM_MISSING, "program missing"},
		{END_CODE_RELATIONAL_ERROR, "relational error"},
		{END_CODE_DUP_DATA_ACCESS, "duplicate data access"},
		{END_CODE_RESPONSE_TOO_BIG, "response too big"},
		{END_CODE_PARAMETER_ERROR, "parameter error"},
		{END_CODE_READ_PROTECTED, "read not possible: protected"},
		{END_CODE_READ_TABLE_MISSING, "read not possible: table missing"},
		{END_CODE_READ_DATA_MISSING, "read not possible: data missing"},
		{END_CODE_READ_PROG_MISSING, "read not possible: program missing"},
		{END_CODE_READ_FILE_MISSING, "read not possible: file missing"},
		{END_CODE_READ_DATA_MISMATCH, "read not possible: data mismatch"},
		{END_CODE_WRITE_READ_ONLY, "write not possible: read only"},
		{END_CODE_WRITE_PROTECTED, "write not possible: write protected"},
		{END_CODE_WRITE_CANT_REGISTER, "write not possible: cannot register"},
		{END_CODE_WRITE_PROG_MISSING, "write not possible: program missing"},
		{END_CODE_WRITE_FILE_MISSING, "write not possible: file missing"},
		{END_CODE_WRITE_FILE_EXISTS, "write not possible: file name already exists"},
		{END_CODE_WRITE_CANT_CHANGE, "write not possible: cannot change"},
		{END_CODE_NOT_EXEC_DURING_EXEC, "not executable in current mode: during execution"},
		{END_CODE_NOT_EXEC_WHILE_RUNNING, "not executable in current mode: while running"},
		{END_CODE_NOT_EXEC_PROG_MODE, "not executable in current mode: PROGRAM mode"},
		{END_CODE_NOT_EXEC_DEBUG_MODE, "not executable in current mode: DEBUG mode"},
		{END_CODE_NOT_EXEC_MONITOR_MODE, "not executable in current mode: MONITOR mode"},
		{END_CODE_NOT_EXEC_RUN_MODE, "not executable in current mode: RUN mode"},
		{END_CODE_NOT_EXEC_NOT_POLLING, "not executable in current mode: not polling node"},
		{END_CODE_NOT_EXEC_STEP_CANT, "not executable in current mode: step cannot be executed"},
		{END_CODE_NO_SUCH_DEVICE_FILE, "no such device: file device missing"},
		{END_CODE_NO_SUCH_DEVICE_MEM, "no such device: memory missing"},
		{END_CODE_NO_SUCH_DEVICE_CLOCK, "no such device: clock missing"},
		{END_CODE_CANT_START_STOP_TABLE, "cannot start/stop: table missing"},
		{END_CODE_UNIT_ERROR_MEMORY, "unit error: memory error"},
		{END_CODE_UNIT_ERROR_IO, "unit error: IO error"},
		{END_CODE_UNIT_ERROR_TOO_MANY_IO, "unit error: too many IO points"},
		{END_CODE_UNIT_ERROR_CPU_BUS, "unit error: CPU bus error"},
		{END_CODE_UNIT_ERROR_IO_DUP, "unit error: IO duplication"},
		{END_CODE_UNIT_ERROR_IO_BUS_ERR, "unit error: IO bus error"},
		{END_CODE_UNIT_ERROR_SYSMAC_BUS2, "unit error: SYSMAC BUS/2 error"},
		{END_CODE_UNIT_ERROR_CPU_BUS_UNIT, "unit error: CPU bus unit error"},
		{END_CODE_UNIT_ERROR_SYSMAC_BUS_NUM_DUP, "unit error: SYSMAC bus number duplication"},
		{END_CODE_UNIT_ERROR_MEM_STATUS, "unit error: memory status error"},
		{END_CODE_UNIT_ERROR_SYSMAC_BUS_TERM, "unit error: SYSMAC bus terminator missing"},
		{END_CODE_CMD_ERROR_NO_PROTECTION, "command error: no protection"},
		{END_CODE_CMD_ERROR_BAD_PASSWORD, "command error: incorrect password"},
		{END_CODE_CMD_ERROR_PROTECTED, "command error: protected"},
		{END_CODE_CMD_ERROR_SERVICE_EXEC, "command error: service already executing"},
		{END_CODE_CMD_ERROR_SERVICE_STOPPED, "command error: service stopped"},
		{END_CODE_CMD_ERROR_NO_EXEC_RIGHT, "command error: no execution right"},
		{END_CODE_CMD_ERROR_SETTINGS_NOT_COMPLETE, "command error: settings not complete"},
		{END_CODE_CMD_ERROR_ITEMS_NOT_SET, "command error: necessary items not set"},
		{END_CODE_CMD_ERROR_NUM_ALREADY_DEFINED, "command error: number already defined"},
		{END_CODE_CMD_ERROR_ERROR_WONT_CLEAR, "command error: error will not clear"},
		{END_CODE_ACCESS_WRITE_NO_RIGHT, "access write error: no access right"},
		{END_CODE_ABORT_SERVICE_ABORTED, "abort: service aborted"},
		{0xFFFF, "unknown error"},
	}

	for _, tt := range tests {
		got := EndCodeText(tt.code)
		if got != tt.want {
			t.Errorf("EndCodeText(%04x) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestEncodeReadCommand(t *testing.T) {
	addr := MemoryAddress{Area: MemoryAreaDMWord, Address: 100, BitOffset: 0}
	cmd := EncodeReadCommand(addr, 10)

	if len(cmd) != 8 {
		t.Fatalf("expected length 8, got %d", len(cmd))
	}

	cmdCode := binary.BigEndian.Uint16(cmd[0:2])
	if cmdCode != MRC_MEMORY_AREA_READ {
		t.Errorf("expected command %04x, got %04x", MRC_MEMORY_AREA_READ, cmdCode)
	}

	count := binary.BigEndian.Uint16(cmd[6:8])
	if count != 10 {
		t.Errorf("expected count 10, got %d", count)
	}
}

func TestEncodeWriteCommand(t *testing.T) {
	addr := MemoryAddress{Area: MemoryAreaDMWord, Address: 100, BitOffset: 0}
	data := []byte{0x00, 0x01, 0x00, 0x02}
	cmd := EncodeWriteCommand(addr, 2, data)

	if len(cmd) != 8+len(data) {
		t.Fatalf("expected length %d, got %d", 8+len(data), len(cmd))
	}

	cmdCode := binary.BigEndian.Uint16(cmd[0:2])
	if cmdCode != MRC_MEMORY_AREA_WRITE {
		t.Errorf("expected command %04x, got %04x", MRC_MEMORY_AREA_WRITE, cmdCode)
	}
}

func TestNewMemoryAddress(t *testing.T) {
	addr := NewMemoryAddress(MemoryAreaDMWord, 200)
	if addr.Area != MemoryAreaDMWord {
		t.Errorf("expected area %02x, got %02x", MemoryAreaDMWord, addr.Area)
	}
	if addr.Address != 200 {
		t.Errorf("expected address 200, got %d", addr.Address)
	}
	if addr.BitOffset != 0 {
		t.Errorf("expected bit offset 0, got %d", addr.BitOffset)
	}
}

func TestNewMemoryAddressWithBit(t *testing.T) {
	addr := NewMemoryAddressWithBit(MemoryAreaDMBit, 200, 3)
	if addr.Area != MemoryAreaDMBit {
		t.Errorf("expected area %02x, got %02x", MemoryAreaDMBit, addr.Area)
	}
	if addr.Address != 200 {
		t.Errorf("expected address 200, got %d", addr.Address)
	}
	if addr.BitOffset != 3 {
		t.Errorf("expected bit offset 3, got %d", addr.BitOffset)
	}
}
