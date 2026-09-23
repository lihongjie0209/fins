package fins

import (
	"reflect"
	"testing"
)

func TestStrictMemorySelectorRoundTrip(t *testing.T) {
	tests := []struct {
		text string
		code byte
		word uint16
		bit  int
	}{
		{"CIO100", 0xb0, 100, -1}, {"CIO100.15", 0x30, 100, 15},
		{"WR20", 0xb1, 20, -1}, {"HR30.1", 0x32, 30, 1},
		{"AR0", 0xb3, 0, -1}, {"DM65535", 0x82, 65535, -1},
		{"EM0:100", 0xa0, 100, -1}, {"EMF:100.2", 0x2f, 100, 2},
		{"EM10:1", 0x60, 1, -1}, {"EM18:2.3", 0xe8, 2, 3},
	}
	for _, test := range tests {
		selector, err := ParseStrictMemorySelector(test.text)
		if err != nil || selector.Code != test.code || selector.Word != test.word || selector.Bit != test.bit || selector.String() != test.text {
			t.Fatalf("%s: selector=%#v string=%q err=%v", test.text, selector, selector.String(), err)
		}
	}
	for _, invalid := range []string{"", "D0", "dm0", "CIO01", "CIO0.16", "EM19:0", "EMG:0", "DM65536", "WR1."} {
		if _, err := ParseStrictMemorySelector(invalid); err == nil {
			t.Fatalf("accepted %q", invalid)
		}
	}
}

func TestStrictMemoryReadWriteCodecs(t *testing.T) {
	word, _ := ParseStrictMemorySelector("DM100")
	read, err := BuildStrictMemoryRead(word, 2)
	if err != nil || read.Main != 1 || read.Sub != 1 || !reflect.DeepEqual(read.Data, []byte{0x82, 0, 100, 0, 0, 2}) {
		t.Fatalf("read=%#v err=%v", read, err)
	}
	words, err := DecodeStrictMemoryWords([]byte{0x12, 0x34, 0xff, 0xfe}, word, 2)
	if err != nil || !reflect.DeepEqual(words, []uint16{0x1234, 0xfffe}) {
		t.Fatalf("words=%#v err=%v", words, err)
	}
	write, err := BuildStrictMemoryWordWrite(word, []uint16{0x1234, 0xfffe})
	if err != nil || write.Sub != 2 || !reflect.DeepEqual(write.Data[6:], []byte{0x12, 0x34, 0xff, 0xfe}) {
		t.Fatalf("write=%#v err=%v", write, err)
	}
	bit, _ := ParseStrictMemorySelector("CIO10.3")
	bitWrite, err := BuildStrictMemoryBitWrite(bit, []bool{true, false, true})
	if err != nil || !reflect.DeepEqual(bitWrite.Data, []byte{0x30, 0, 10, 3, 0, 3, 1, 0, 1}) {
		t.Fatalf("bit write=%#v err=%v", bitWrite, err)
	}
	bits, err := DecodeStrictMemoryBits([]byte{1, 0, 1}, bit, 3)
	if err != nil || !reflect.DeepEqual(bits, []bool{true, false, true}) {
		t.Fatalf("bits=%#v err=%v", bits, err)
	}
}

func TestStrictMemoryRejectsTypesBoundsAndResponses(t *testing.T) {
	word, _ := ParseStrictMemorySelector("DM0")
	bit, _ := ParseStrictMemorySelector("DM0.0")
	if _, err := BuildStrictMemoryRead(word, 0); err == nil {
		t.Fatal("accepted zero read")
	}
	if _, err := BuildStrictMemoryWordWrite(bit, []uint16{1}); err == nil {
		t.Fatal("accepted word write to bit")
	}
	if _, err := BuildStrictMemoryBitWrite(word, []bool{true}); err == nil {
		t.Fatal("accepted bit write to word")
	}
	if _, err := DecodeStrictMemoryWords([]byte{1}, word, 1); err == nil {
		t.Fatal("accepted short word response")
	}
	if _, err := DecodeStrictMemoryBits([]byte{2}, bit, 1); err == nil {
		t.Fatal("accepted invalid bit response")
	}
	lastWord, _ := ParseStrictMemorySelector("DM65535")
	if err := ValidateStrictMemoryRange(lastWord, 2); err == nil {
		t.Fatal("accepted overflowing word range")
	}
	lastBit, _ := ParseStrictMemorySelector("DM65535.15")
	if _, err := BuildStrictMemoryRead(lastBit, 2); err == nil {
		t.Fatal("accepted overflowing bit range")
	}
}
