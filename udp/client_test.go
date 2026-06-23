package udp

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFinsClient(t *testing.T) {
	clientAddr := NewAddress("", 9600, 0, 2, 0)
	plcAddr := NewAddress("", 9601, 0, 10, 0)

	toWrite := []uint16{5, 4, 3, 2, 1}

	s, e := NewPLCSimulator(plcAddr)
	if e != nil {
		panic(e)
	}
	defer s.Close()

	c, e := NewClient(clientAddr, plcAddr)
	if e != nil {
		panic(e)
	}
	defer c.Close()

	// ------------- Test Words
	err := c.WriteWords(MemoryAreaDMWord, 100, toWrite)
	assert.Nil(t, err)

	vals, err := c.ReadWords(MemoryAreaDMWord, 100, 5)
	assert.Nil(t, err)
	assert.Equal(t, toWrite, vals)

	// test setting response timeout
	c.SetTimeoutMs(50)
	_, err = c.ReadWords(MemoryAreaDMWord, 100, 5)
	assert.Nil(t, err)

	// ------------- Test Strings
	err = c.WriteString(MemoryAreaDMWord, 10, "ф1234")
	assert.Nil(t, err)

	v, err := c.ReadString(MemoryAreaDMWord, 12, 1)
	assert.Nil(t, err)
	assert.Equal(t, "12", v)

	v, err = c.ReadString(MemoryAreaDMWord, 10, 3)
	assert.Nil(t, err)
	assert.Equal(t, "ф1234", v)

	v, err = c.ReadString(MemoryAreaDMWord, 10, 5)
	assert.Nil(t, err)
	assert.Equal(t, "ф1234", v)

	// ------------- Test Bytes
	err = c.WriteBytes(MemoryAreaDMWord, 10, []byte{0x00, 0x00 ,0xC1 , 0xA0})
	assert.Nil(t, err)

	b, err := c.ReadBytes(MemoryAreaDMWord, 10, 2)
	assert.Nil(t, err)
	assert.Equal(t, []byte{0x00, 0x00 ,0xC1 , 0xA0}, b)

	buf := make([]byte, 8, 8)
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(-20))
	err = c.WriteBytes(MemoryAreaDMWord, 10, buf)
	assert.Nil(t, err)

	b, err = c.ReadBytes(MemoryAreaDMWord, 10, 4)
	assert.Nil(t, err)
	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x34, 0xc0}, b)


	// ------------- Test Bits
	err = c.WriteBits(MemoryAreaDMBit, 10, 2, []bool{true, false, true})
	assert.Nil(t, err)

	bs, err := c.ReadBits(MemoryAreaDMBit, 10, 2, 3)
	assert.Nil(t, err)
	assert.Equal(t, []bool{true, false, true}, bs)

	bs, err = c.ReadBits(MemoryAreaDMBit, 10, 1, 5)
	assert.Nil(t, err)
	assert.Equal(t, []bool{false, true, false, true, false}, bs)

}

func TestFinsClient_SetGetBit(t *testing.T) {
	clientAddr := NewAddress("", 9602, 0, 2, 0)
	plcAddr := NewAddress("", 9603, 0, 10, 0)

	s, e := NewPLCSimulator(plcAddr)
	if e != nil {
		panic(e)
	}
	defer s.Close()

	c, e := NewClient(clientAddr, plcAddr)
	if e != nil {
		panic(e)
	}
	defer c.Close()

	err := c.SetBit(MemoryAreaDMBit, 200, 5)
	assert.Nil(t, err)

	bs, err := c.ReadBits(MemoryAreaDMBit, 200, 5, 1)
	assert.Nil(t, err)
	assert.Equal(t, true, bs[0])

	err = c.ResetBit(MemoryAreaDMBit, 200, 5)
	assert.Nil(t, err)

	bs, err = c.ReadBits(MemoryAreaDMBit, 200, 5, 1)
	assert.Nil(t, err)
	assert.Equal(t, false, bs[0])

	err = c.ToggleBit(MemoryAreaDMBit, 200, 5)
	assert.Nil(t, err)

	bs, err = c.ReadBits(MemoryAreaDMBit, 200, 5, 1)
	assert.Nil(t, err)
	assert.Equal(t, true, bs[0])
}

func TestFinsClient_SetByteOrder(t *testing.T) {
	clientAddr := NewAddress("", 9604, 0, 2, 0)
	plcAddr := NewAddress("", 9605, 0, 10, 0)

	s, e := NewPLCSimulator(plcAddr)
	if e != nil {
		panic(e)
	}
	defer s.Close()

	c, e := NewClient(clientAddr, plcAddr)
	if e != nil {
		panic(e)
	}
	defer c.Close()

	c.SetByteOrder(binary.LittleEndian)

	toWrite := []uint16{0x1234}
	err := c.WriteWords(MemoryAreaDMWord, 300, toWrite)
	assert.Nil(t, err)

	vals, err := c.ReadWords(MemoryAreaDMWord, 300, 1)
	assert.Nil(t, err)
	assert.Equal(t, toWrite, vals)
}

func TestBCDEncoding(t *testing.T) {
	testCases := []struct {
		input    uint64
		expected []byte
	}{
		{0, []byte{0x0f}},
		{1, []byte{0x1f}},
		{12, []byte{0x12}},
		{123, []byte{0x12, 0x3f}},
		{1234, []byte{0x12, 0x34}},
		{12345, []byte{0x12, 0x34, 0x5f}},
	}

	for _, tc := range testCases {
		result := encodeBCD(tc.input)
		assert.Equal(t, tc.expected, result, "encodeBCD(%d)", tc.input)
	}
}

func TestBCDDecoding(t *testing.T) {
	testCases := []struct {
		input    []byte
		expected uint64
		err      bool
	}{
		{[]byte{0x0f}, 0, false},
		{[]byte{0x1f}, 1, false},
		{[]byte{0x12}, 12, false},
		{[]byte{0x12, 0x3f}, 123, false},
		{[]byte{0x12, 0x34}, 1234, false},
		{[]byte{0x12, 0x34, 0x5f}, 12345, false},
		{[]byte{0x1a}, 0, true},
		{[]byte{0xa1}, 0, true},
		{[]byte{0x12, 0xaf}, 0, true},
	}

	for _, tc := range testCases {
		result, err := decodeBCD(tc.input)
		if tc.err {
			assert.Error(t, err, "decodeBCD(%v) should error", tc.input)
		} else {
			assert.Nil(t, err, "decodeBCD(%v) should not error", tc.input)
			assert.Equal(t, tc.expected, result, "decodeBCD(%v)", tc.input)
		}
	}
}

func TestCheckIsWordMemoryArea(t *testing.T) {
	assert.True(t, checkIsWordMemoryArea(MemoryAreaDMWord))
	assert.True(t, checkIsWordMemoryArea(MemoryAreaARWord))
	assert.True(t, checkIsWordMemoryArea(MemoryAreaHRWord))
	assert.True(t, checkIsWordMemoryArea(MemoryAreaWRWord))
	assert.False(t, checkIsWordMemoryArea(MemoryAreaDMBit))
	assert.False(t, checkIsWordMemoryArea(0xFF))
}

func TestCheckIsBitMemoryArea(t *testing.T) {
	assert.True(t, checkIsBitMemoryArea(MemoryAreaDMBit))
	assert.True(t, checkIsBitMemoryArea(MemoryAreaARBit))
	assert.True(t, checkIsBitMemoryArea(MemoryAreaHRBit))
	assert.True(t, checkIsBitMemoryArea(MemoryAreaWRBit))
	assert.False(t, checkIsBitMemoryArea(MemoryAreaDMWord))
	assert.False(t, checkIsBitMemoryArea(0xFF))
}

func TestClockReadCommand(t *testing.T) {
	cmd := clockReadCommand()
	assert.NotEmpty(t, cmd)
}

func TestTimesTenPlusCatchingOverflow(t *testing.T) {
	result, err := timesTenPlusCatchingOverflow(123, 4)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1234), result)

	_, err = timesTenPlusCatchingOverflow(math.MaxUint64, 1)
	assert.Error(t, err)
}

func TestFinsClient_ReadWriteBitWithInvalidArea(t *testing.T) {
	clientAddr := NewAddress("", 9610, 0, 2, 0)
	plcAddr := NewAddress("", 9611, 0, 10, 0)

	s, e := NewPLCSimulator(plcAddr)
	if e != nil {
		panic(e)
	}
	defer s.Close()

	c, e := NewClient(clientAddr, plcAddr)
	if e != nil {
		panic(e)
	}
	defer c.Close()

	_, err := c.ReadWords(MemoryAreaDMBit, 100, 1)
	assert.Error(t, err)

	err = c.WriteWords(MemoryAreaDMBit, 100, []uint16{123})
	assert.Error(t, err)

	_, err = c.ReadBytes(MemoryAreaDMBit, 100, 1)
	assert.Error(t, err)

	err = c.WriteBytes(MemoryAreaDMBit, 100, []byte{0x01, 0x02})
	assert.Error(t, err)

	_, err = c.ReadBits(MemoryAreaDMWord, 100, 0, 1)
	assert.Error(t, err)

	err = c.WriteBits(MemoryAreaDMWord, 100, 0, []bool{true})
	assert.Error(t, err)

	err = c.SetBit(MemoryAreaDMWord, 100, 5)
	assert.Error(t, err)

	err = c.ResetBit(MemoryAreaDMWord, 100, 5)
	assert.Error(t, err)

	err = c.ToggleBit(MemoryAreaDMWord, 100, 5)
	assert.Error(t, err)
}

func TestResponseTimeoutError(t *testing.T) {
	err := ResponseTimeoutError{50}
	assert.Contains(t, err.Error(), "50")
}

func TestBCDBadDigitError(t *testing.T) {
	err := BCDBadDigitError{"hi", 15}
	assert.Contains(t, err.Error(), "hi")
	assert.Contains(t, err.Error(), "15")
}

func TestCheckResponse(t *testing.T) {
	err := checkResponse(&response{endCode: EndCodeNormalCompletion}, nil)
	assert.Nil(t, err)

	err = checkResponse(&response{endCode: 0x0101}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "0x101")

	testErr := assert.AnError
	err = checkResponse(nil, testErr)
	assert.Error(t, err)
}

func TestBCDOverflowError(t *testing.T) {
	err := BCDOverflowError{}
	assert.Contains(t, err.Error(), "Overflow")
}

func TestFinsClient_ReadClock(t *testing.T) {
	clientAddr := NewAddress("", 9606, 0, 2, 0)
	plcAddr := NewAddress("", 9607, 0, 10, 0)

	s, e := NewPLCSimulator(plcAddr)
	if e != nil {
		panic(e)
	}
	defer s.Close()

	c, e := NewClient(clientAddr, plcAddr)
	if e != nil {
		panic(e)
	}
	defer c.Close()

	c.SetTimeoutMs(500)
	_, err := c.ReadClock()
	assert.Nil(t, err)
}
