package fins

import (
	"math"
	"testing"
)

func TestDecoder_NewDecoder(t *testing.T) {
	d := NewDecoder()
	if d == nil {
		t.Fatal("expected non-nil decoder")
	}
	if d.areaMap == nil {
		t.Fatal("expected non-nil area map")
	}
}

func TestDecoder_GetAreaInfo(t *testing.T) {
	d := NewDecoder()

	tests := []struct {
		area     string
		expected bool
	}{
		{"CIO", true},
		{"W", true},
		{"H", true},
		{"A", true},
		{"D", true},
		{"P", true},
		{"F", true},
		{"EM0", true},
		{"EM15", true},
		{"EM16", false},
		{"INVALID", false},
	}

	for _, tt := range tests {
		_, ok := d.GetAreaInfo(tt.area)
		if ok != tt.expected {
			t.Errorf("GetAreaInfo(%q) ok = %v, want %v", tt.area, ok, tt.expected)
		}
	}
}

func TestDecoder_GetAreaCode(t *testing.T) {
	d := NewDecoder()

	code, err := d.GetAreaCode("D")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != MemoryAreaDMWord {
		t.Errorf("expected %02x, got %02x", MemoryAreaDMWord, code)
	}

	_, err = d.GetAreaCode("INVALID")
	if err == nil {
		t.Error("expected error for invalid area")
	}
}

func TestDecoder_ParseAddress_Word(t *testing.T) {
	d := NewDecoder()

	tests := []struct {
		addr     string
		area     string
		address  int
		isBit    bool
		bit      int
		isString bool
		strLen   int
	}{
		{"D100", "D", 100, false, -1, false, 0},
		{"CIO0", "CIO", 0, false, -1, false, 0},
		{"W500", "W", 500, false, -1, false, 0},
		{"H10", "H", 10, false, -1, false, 0},
		{"A200", "A", 200, false, -1, false, 0},
		{"EM10W100", "EM10", 100, false, -1, false, 0},
		{"EM0W0", "EM0", 0, false, -1, false, 0},
		{"EM15W500", "EM15", 500, false, -1, false, 0},
	}

	for _, tt := range tests {
		parsed, err := d.ParseAddress(tt.addr)
		if err != nil {
			t.Fatalf("ParseAddress(%q) error: %v", tt.addr, err)
		}
		if parsed.Area != tt.area {
			t.Errorf("ParseAddress(%q) area = %q, want %q", tt.addr, parsed.Area, tt.area)
		}
		if parsed.Address != tt.address {
			t.Errorf("ParseAddress(%q) address = %d, want %d", tt.addr, parsed.Address, tt.address)
		}
		if parsed.IsBit != tt.isBit {
			t.Errorf("ParseAddress(%q) isBit = %v, want %v", tt.addr, parsed.IsBit, tt.isBit)
		}
		if parsed.Bit != tt.bit {
			t.Errorf("ParseAddress(%q) bit = %d, want %d", tt.addr, parsed.Bit, tt.bit)
		}
		if parsed.IsString != tt.isString {
			t.Errorf("ParseAddress(%q) isString = %v, want %v", tt.addr, parsed.IsString, tt.isString)
		}
		if parsed.StringLen != tt.strLen {
			t.Errorf("ParseAddress(%q) stringLen = %d, want %d", tt.addr, parsed.StringLen, tt.strLen)
		}
	}
}

func TestDecoder_ParseAddress_Bit(t *testing.T) {
	d := NewDecoder()

	tests := []struct {
		addr    string
		area    string
		address int
		bit     int
	}{
		{"CIO0.0", "CIO", 0, 0},
		{"CIO100.15", "CIO", 100, 15},
		{"D10.5", "D", 10, 5},
		{"W20.3", "W", 20, 3},
		{"H5.10", "H", 5, 10},
		{"EM5W100.7", "EM5", 100, 7},
	}

	for _, tt := range tests {
		parsed, err := d.ParseAddress(tt.addr)
		if err != nil {
			t.Fatalf("ParseAddress(%q) error: %v", tt.addr, err)
		}
		if !parsed.IsBit {
			t.Errorf("ParseAddress(%q) expected isBit=true", tt.addr)
		}
		if parsed.Area != tt.area {
			t.Errorf("ParseAddress(%q) area = %q, want %q", tt.addr, parsed.Area, tt.area)
		}
		if parsed.Address != tt.address {
			t.Errorf("ParseAddress(%q) address = %d, want %d", tt.addr, parsed.Address, tt.address)
		}
		if parsed.Bit != tt.bit {
			t.Errorf("ParseAddress(%q) bit = %d, want %d", tt.addr, parsed.Bit, tt.bit)
		}
	}
}

func TestDecoder_ParseAddress_String(t *testing.T) {
	d := NewDecoder()

	tests := []struct {
		addr      string
		strLen    int
		byteOrder ByteOrder
	}{
		{"D0.20", 20, ByteOrderLow},
		{"D0.20L", 20, ByteOrderLow},
		{"D10.50H", 50, ByteOrderHigh},
		{"CIO100.10H", 10, ByteOrderHigh},
	}

	for _, tt := range tests {
		parsed, err := d.ParseAddress(tt.addr)
		if err != nil {
			t.Fatalf("ParseAddress(%q) error: %v", tt.addr, err)
		}
		if !parsed.IsString {
			t.Errorf("ParseAddress(%q) expected isString=true", tt.addr)
		}
		if parsed.StringLen != tt.strLen {
			t.Errorf("ParseAddress(%q) stringLen = %d, want %d", tt.addr, parsed.StringLen, tt.strLen)
		}
		if parsed.ByteOrder != tt.byteOrder {
			t.Errorf("ParseAddress(%q) byteOrder = %d, want %d", tt.addr, parsed.ByteOrder, tt.byteOrder)
		}
	}
}

func TestDecoder_ParseAddress_Invalid(t *testing.T) {
	d := NewDecoder()

	invalidAddrs := []string{
		"",
		"INVALID",
		"X100",
		"D",
		"CIO.5",
		"CIO0.16",
		"D-1",
		"EM16W100",
		"F0.10",
	}

	for _, addr := range invalidAddrs {
		_, err := d.ParseAddress(addr)
		if err == nil {
			t.Errorf("ParseAddress(%q) expected error", addr)
		}
	}
}

func TestDecoder_EncodeDecode_UINT16(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	encoded, err := d.EncodeValue(uint16(1234), DataTypeUINT16, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 2 {
		t.Fatalf("expected 2 bytes, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeUINT16, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(uint16)
	if !ok {
		t.Fatalf("expected uint16, got %T", decoded)
	}

	if val != 1234 {
		t.Errorf("expected 1234, got %d", val)
	}
}

func TestDecoder_EncodeDecode_INT32(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	encoded, err := d.EncodeValue(int32(-123456), DataTypeINT32, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeINT32, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(int32)
	if !ok {
		t.Fatalf("expected int32, got %T", decoded)
	}

	if val != -123456 {
		t.Errorf("expected -123456, got %d", val)
	}
}

func TestDecoder_EncodeDecode_FLOAT(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	testVal := float32(3.14)
	encoded, err := d.EncodeValue(testVal, DataTypeFLOAT, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeFLOAT, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", decoded)
	}

	if math.Abs(float64(val-testVal)) > 0.0001 {
		t.Errorf("expected %f, got %f", testVal, val)
	}
}

func TestDecoder_EncodeDecode_DOUBLE(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	testVal := 3.1415926535
	encoded, err := d.EncodeValue(testVal, DataTypeDOUBLE, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 8 {
		t.Fatalf("expected 8 bytes, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeDOUBLE, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", decoded)
	}

	if math.Abs(val-testVal) > 0.0000001 {
		t.Errorf("expected %f, got %f", testVal, val)
	}
}

func TestDecoder_EncodeDecode_String(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord, IsString: true, StringLen: 20, ByteOrder: ByteOrderHigh}

	testStr := "Hello, World!"
	encoded, err := d.EncodeValue(testStr, DataTypeSTRING, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	decoded, err := d.DecodeValue(encoded, DataTypeSTRING, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(string)
	if !ok {
		t.Fatalf("expected string, got %T", decoded)
	}

	if val != testStr {
		t.Errorf("expected %q, got %q", testStr, val)
	}
}

func TestDecoder_EncodeDecode_Bit(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMBit, IsBit: true, Bit: 3}

	encoded, err := d.EncodeValue(true, DataTypeBIT, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 1 {
		t.Fatalf("expected 1 byte, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeBIT, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T", decoded)
	}

	if !val {
		t.Errorf("expected true, got false")
	}
}

func TestDecoder_EncodeDecode_UINT8(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaFlagBit}

	testVal := uint8(0xAB)
	encoded, err := d.EncodeValue(testVal, DataTypeUINT8, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 1 {
		t.Fatalf("expected 1 byte, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeUINT8, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(uint8)
	if !ok {
		t.Fatalf("expected uint8, got %T", decoded)
	}

	if val != testVal {
		t.Errorf("expected %d, got %d", testVal, val)
	}
}

func TestDecoder_EncodeDecode_INT64(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	testVal := int64(-9876543210)
	encoded, err := d.EncodeValue(testVal, DataTypeINT64, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 8 {
		t.Fatalf("expected 8 bytes, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeINT64, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", decoded)
	}

	if val != testVal {
		t.Errorf("expected %d, got %d", testVal, val)
	}
}

func TestDecoder_DataTypeSize(t *testing.T) {
	d := NewDecoder()

	tests := []struct {
		dataType DataType
		expected int
	}{
		{DataTypeUINT8, 1},
		{DataTypeINT8, 1},
		{DataTypeUINT16, 2},
		{DataTypeINT16, 2},
		{DataTypeUINT32, 4},
		{DataTypeINT32, 4},
		{DataTypeFLOAT, 4},
		{DataTypeUINT64, 8},
		{DataTypeINT64, 8},
		{DataTypeDOUBLE, 8},
		{DataTypeSTRING, 0},
	}

	for _, tt := range tests {
		size, err := d.DataTypeSize(tt.dataType)
		if err != nil {
			t.Fatalf("DataTypeSize(%s) error: %v", tt.dataType, err)
		}
		if size != tt.expected {
			t.Errorf("DataTypeSize(%s) = %d, want %d", tt.dataType, size, tt.expected)
		}
	}
}

func TestDecoder_Decode_DataTooShort(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	shortData := []byte{0x01}
	_, err := d.DecodeValue(shortData, DataTypeUINT16, parsed)
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestDecoder_IsBitDataType(t *testing.T) {
	d := NewDecoder()

	if !d.IsBitDataType(DataTypeBIT) {
		t.Error("BIT should be bit data type")
	}
	if d.IsBitDataType(DataTypeUINT16) {
		t.Error("UINT16 should not be bit data type")
	}
}

func TestGetWordAreaCode(t *testing.T) {
	tests := []struct {
		bitCode  uint8
		wordCode uint8
		err      bool
	}{
		{MemoryAreaCIOBit, MemoryAreaCIOWord, false},
		{MemoryAreaDMBit, MemoryAreaDMWord, false},
		{MemoryAreaWRBit, MemoryAreaWRWord, false},
		{MemoryAreaHRBit, MemoryAreaHRWord, false},
		{MemoryAreaEM5Bit, MemoryAreaEM5Word, false},
		{0xFF, 0, true},
	}

	for _, tt := range tests {
		got, err := GetWordAreaCode(tt.bitCode)
		if (err != nil) != tt.err {
			t.Errorf("GetWordAreaCode(%02x) error = %v, wantErr %v", tt.bitCode, err, tt.err)
			continue
		}
		if !tt.err && got != tt.wordCode {
			t.Errorf("GetWordAreaCode(%02x) = %02x, want %02x", tt.bitCode, got, tt.wordCode)
		}
	}
}

func TestGetBitAreaCode(t *testing.T) {
	tests := []struct {
		wordCode uint8
		bitCode  uint8
		err      bool
	}{
		{MemoryAreaCIOWord, MemoryAreaCIOBit, false},
		{MemoryAreaDMWord, MemoryAreaDMBit, false},
		{MemoryAreaWRWord, MemoryAreaWRBit, false},
		{MemoryAreaHRWord, MemoryAreaHRBit, false},
		{MemoryAreaEM10Word, MemoryAreaEM10Bit, false},
		{0xFF, 0, true},
	}

	for _, tt := range tests {
		got, err := GetBitAreaCode(tt.wordCode)
		if (err != nil) != tt.err {
			t.Errorf("GetBitAreaCode(%02x) error = %v, wantErr %v", tt.wordCode, err, tt.err)
			continue
		}
		if !tt.err && got != tt.bitCode {
			t.Errorf("GetBitAreaCode(%02x) = %02x, want %02x", tt.wordCode, got, tt.bitCode)
		}
	}
}

func TestDecoder_EncodeValue_Float64Conversion(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	encoded, err := d.EncodeValue(float64(3.14), DataTypeFLOAT, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	decoded, err := d.DecodeValue(encoded, DataTypeFLOAT, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(float32)
	if !ok {
		t.Fatalf("expected float32, got %T", decoded)
	}

	if math.Abs(float64(val-3.14)) > 0.001 {
		t.Errorf("expected ~3.14, got %f", val)
	}
}

func TestDecoder_DecodeValue_NullTerminatedString(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord, IsString: true, StringLen: 20}

	data := make([]byte, 20)
	copy(data, "Hello\x00World")

	decoded, err := d.DecodeValue(data, DataTypeSTRING, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(string)
	if !ok {
		t.Fatalf("expected string, got %T", decoded)
	}

	if val != "Hello" {
		t.Errorf("expected 'Hello', got %q", val)
	}
}

func TestDecoder_EncodeDecode_INT8(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	testVal := int8(-123)
	encoded, err := d.EncodeValue(testVal, DataTypeINT8, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 1 {
		t.Fatalf("expected 1 byte, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeINT8, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(int8)
	if !ok {
		t.Fatalf("expected int8, got %T", decoded)
	}

	if val != testVal {
		t.Errorf("expected %d, got %d", testVal, val)
	}
}

func TestDecoder_EncodeDecode_INT16(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	testVal := int16(-12345)
	encoded, err := d.EncodeValue(testVal, DataTypeINT16, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 2 {
		t.Fatalf("expected 2 bytes, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeINT16, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(int16)
	if !ok {
		t.Fatalf("expected int16, got %T", decoded)
	}

	if val != testVal {
		t.Errorf("expected %d, got %d", testVal, val)
	}
}

func TestDecoder_EncodeDecode_UINT32(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	testVal := uint32(3123456789)
	encoded, err := d.EncodeValue(testVal, DataTypeUINT32, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 4 {
		t.Fatalf("expected 4 bytes, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeUINT32, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(uint32)
	if !ok {
		t.Fatalf("expected uint32, got %T", decoded)
	}

	if val != testVal {
		t.Errorf("expected %d, got %d", testVal, val)
	}
}

func TestDecoder_EncodeDecode_UINT64(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	testVal := uint64(123456789012345)
	encoded, err := d.EncodeValue(testVal, DataTypeUINT64, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	if len(encoded) != 8 {
		t.Fatalf("expected 8 bytes, got %d", len(encoded))
	}

	decoded, err := d.DecodeValue(encoded, DataTypeUINT64, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(uint64)
	if !ok {
		t.Fatalf("expected uint64, got %T", decoded)
	}

	if val != testVal {
		t.Errorf("expected %d, got %d", testVal, val)
	}
}

func TestDecoder_EncodeValue_Float64ToDouble(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	encoded, err := d.EncodeValue(float64(3.14159), DataTypeDOUBLE, parsed)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	decoded, err := d.DecodeValue(encoded, DataTypeDOUBLE, parsed)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	val, ok := decoded.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", decoded)
	}

	if math.Abs(val-3.14159) > 0.0001 {
		t.Errorf("expected ~3.14159, got %f", val)
	}
}

func TestDecoder_DecodeValue_InvalidType(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	_, err := d.DecodeValue([]byte{0x01, 0x02}, "INVALID", parsed)
	if err == nil {
		t.Error("expected error for invalid data type")
	}
}

func TestDecoder_EncodeValue_InvalidType(t *testing.T) {
	d := NewDecoder()
	parsed := &ParsedAddress{AreaCode: MemoryAreaDMWord}

	_, err := d.EncodeValue(123, "INVALID", parsed)
	if err == nil {
		t.Error("expected error for invalid data type")
	}
}
