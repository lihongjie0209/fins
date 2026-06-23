package fins

import (
	"testing"
)

func TestMockPLC_Addr(t *testing.T) {
	mockPLC := NewMockPLC()
	addr, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	if mockPLC.Addr() != addr {
		t.Errorf("expected addr %s, got %s", addr, mockPLC.Addr())
	}
}

func TestMockPLC_GetArea(t *testing.T) {
	mockPLC := NewMockPLC()
	_, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	testCases := []struct {
		areaCode uint8
		expected bool
	}{
		{MemoryAreaDMWord, true},
		{MemoryAreaCIOWord, true},
		{MemoryAreaWRWord, true},
		{MemoryAreaHRWord, true},
		{MemoryAreaARWord, true},
		{0xFF, false},
	}

	for _, tc := range testCases {
		area, _ := mockPLC.getArea(tc.areaCode)
		if (area != nil) != tc.expected {
			t.Errorf("getArea(0x%02x): expected nil=%v, got nil=%v", tc.areaCode, !tc.expected, area == nil)
		}
	}
}

func TestMockPLC_GetBitArea(t *testing.T) {
	mockPLC := NewMockPLC()
	_, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	testCases := []struct {
		areaCode uint8
		expected bool
	}{
		{MemoryAreaDMBit, true},
		{MemoryAreaCIOBit, true},
		{MemoryAreaWRBit, true},
		{MemoryAreaHRBit, true},
		{MemoryAreaARBit, true},
		{0xFF, false},
	}

	for _, tc := range testCases {
		area, _ := mockPLC.getBitArea(tc.areaCode)
		if (area != nil) != tc.expected {
			t.Errorf("getBitArea(0x%02x): expected nil=%v, got nil=%v", tc.areaCode, !tc.expected, area == nil)
		}
	}
}

func TestMockPLC_ReadMemory_AddressRange(t *testing.T) {
	mockPLC := NewMockPLC()
	_, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	addrTooLarge := NewMemoryAddress(MemoryAreaDMWord, uint16(MockDMWordSize+1))
	_, endCode := mockPLC.readMemory(addrTooLarge, 1)
	if endCode != END_CODE_ADDR_RANGE_EXCEED {
		t.Errorf("expected END_CODE_ADDR_RANGE_EXCEED, got %d", endCode)
	}
}

func TestMockPLC_ReadMemory_UnsupportedArea(t *testing.T) {
	mockPLC := NewMockPLC()
	_, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	addrUnsupported := NewMemoryAddress(0xFF, 0)
	_, endCode := mockPLC.readMemory(addrUnsupported, 1)
	if endCode != END_CODE_NOT_SUPPORTED {
		t.Errorf("expected END_CODE_NOT_SUPPORTED, got %d", endCode)
	}
}

func TestMockPLC_WriteMemory_AddressRange(t *testing.T) {
	mockPLC := NewMockPLC()
	_, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	addrTooLarge := NewMemoryAddress(MemoryAreaDMWord, uint16(MockDMWordSize+1))
	endCode := mockPLC.writeMemory(addrTooLarge, 1, []byte{0x00, 0x00})
	if endCode != END_CODE_ADDR_RANGE_EXCEED {
		t.Errorf("expected END_CODE_ADDR_RANGE_EXCEED, got %d", endCode)
	}
}

func TestMockPLC_WriteMemory_UnsupportedArea(t *testing.T) {
	mockPLC := NewMockPLC()
	_, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	addrUnsupported := NewMemoryAddress(0xFF, 0)
	endCode := mockPLC.writeMemory(addrUnsupported, 1, []byte{0x00, 0x01})
	if endCode != END_CODE_NOT_SUPPORTED {
		t.Errorf("expected END_CODE_NOT_SUPPORTED, got %d", endCode)
	}
}

func TestMockPLC_GetWord(t *testing.T) {
	mockPLC := NewMockPLC()
	_, err := mockPLC.Start()
	if err != nil {
		t.Fatalf("failed to start mock PLC: %v", err)
	}
	defer mockPLC.Close()

	mockPLC.SetWord(MemoryAreaDMWord, 100, 12345)
	val := mockPLC.GetWord(MemoryAreaDMWord, 100)
	if val != 12345 {
		t.Errorf("expected 12345, got %d", val)
	}

	val = mockPLC.GetWord(MemoryAreaDMWord, 65535)
	if val != 0 {
		t.Errorf("expected 0 for invalid address, got %d", val)
	}
}
