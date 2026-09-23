package fins

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// StrictMemorySelector is one canonical FINS memory word or bit selector.
type StrictMemorySelector struct {
	Area string
	Bank int
	Word uint16
	Bit  int
	Code byte
}

// ParseStrictMemorySelector parses CIO/WR/HR/AR/DM and EM bank selectors.
func ParseStrictMemorySelector(text string) (StrictMemorySelector, error) {
	selector := StrictMemorySelector{Bank: -1, Bit: -1}
	remainder := ""
	for _, area := range []string{"CIO", "WR", "HR", "AR", "DM"} {
		if strings.HasPrefix(text, area) {
			selector.Area, remainder = area, strings.TrimPrefix(text, area)
			break
		}
	}
	if selector.Area == "" && strings.HasPrefix(text, "EM") {
		selector.Area = "EM"
		bankEnd := strings.IndexByte(text, ':')
		if bankEnd < 3 {
			return selector, fmt.Errorf("invalid FINS memory selector %q", text)
		}
		bankText := text[2:bankEnd]
		bank, err := strconv.ParseUint(bankText, 16, 8)
		if err != nil || bank > 0x18 || bankText != strings.ToUpper(bankText) || len(bankText) > 1 && bankText[0] == '0' {
			return selector, fmt.Errorf("invalid FINS memory selector %q", text)
		}
		selector.Bank, remainder = int(bank), text[bankEnd+1:]
	}
	if selector.Area == "" || remainder == "" {
		return selector, fmt.Errorf("invalid FINS memory selector %q", text)
	}
	wordText := remainder
	if dot := strings.IndexByte(remainder, '.'); dot >= 0 {
		wordText = remainder[:dot]
		bitText := remainder[dot+1:]
		bit, err := strconv.ParseUint(bitText, 10, 4)
		if err != nil || bit > 15 || bitText == "" || len(bitText) > 1 && bitText[0] == '0' {
			return selector, fmt.Errorf("invalid FINS memory selector %q", text)
		}
		selector.Bit = int(bit)
	}
	word, err := strconv.ParseUint(wordText, 10, 16)
	if err != nil || wordText == "" || len(wordText) > 1 && wordText[0] == '0' {
		return selector, fmt.Errorf("invalid FINS memory selector %q", text)
	}
	selector.Word = uint16(word)
	selector.Code, err = strictMemoryAreaCode(selector.Area, selector.Bank, selector.Bit >= 0)
	if err != nil {
		return selector, fmt.Errorf("invalid FINS memory selector %q", text)
	}
	return selector, nil
}

func (selector StrictMemorySelector) String() string {
	prefix := selector.Area
	if selector.Area == "EM" {
		prefix += strings.ToUpper(strconv.FormatInt(int64(selector.Bank), 16)) + ":"
	}
	result := prefix + strconv.Itoa(int(selector.Word))
	if selector.Bit >= 0 {
		result += "." + strconv.Itoa(selector.Bit)
	}
	return result
}

// BuildStrictMemoryRead builds a bounded Memory Area Read command.
func BuildStrictMemoryRead(selector StrictMemorySelector, points int) (StrictCommand, error) {
	if err := validateStrictMemoryRange(selector, points); err != nil {
		return StrictCommand{}, err
	}
	return StrictCommand{Main: 1, Sub: 1, Data: strictMemoryAddressAndCount(selector, points)}, nil
}

// BuildStrictMemoryWordWrite builds a bounded word Memory Area Write command.
func BuildStrictMemoryWordWrite(selector StrictMemorySelector, values []uint16) (StrictCommand, error) {
	if selector.Bit >= 0 {
		return StrictCommand{}, errors.New("FINS word write requires a word selector")
	}
	if err := validateStrictMemoryRange(selector, len(values)); err != nil {
		return StrictCommand{}, err
	}
	data := strictMemoryAddressAndCount(selector, len(values))
	for _, value := range values {
		data = binary.BigEndian.AppendUint16(data, value)
	}
	return StrictCommand{Main: 1, Sub: 2, Data: data}, nil
}

// BuildStrictMemoryBitWrite builds a bounded bit Memory Area Write command.
func BuildStrictMemoryBitWrite(selector StrictMemorySelector, values []bool) (StrictCommand, error) {
	if selector.Bit < 0 {
		return StrictCommand{}, errors.New("FINS bit write requires a bit selector")
	}
	if err := validateStrictMemoryRange(selector, len(values)); err != nil {
		return StrictCommand{}, err
	}
	data := strictMemoryAddressAndCount(selector, len(values))
	for _, value := range values {
		if value {
			data = append(data, 1)
		} else {
			data = append(data, 0)
		}
	}
	return StrictCommand{Main: 1, Sub: 2, Data: data}, nil
}

// DecodeStrictMemoryWords validates exact word response cardinality.
func DecodeStrictMemoryWords(data []byte, selector StrictMemorySelector, points int) ([]uint16, error) {
	if selector.Bit >= 0 {
		return nil, errors.New("FINS word response requires a word selector")
	}
	if err := validateStrictMemoryRange(selector, points); err != nil {
		return nil, err
	}
	if len(data) != points*2 {
		return nil, errors.New("FINS memory response cardinality mismatch")
	}
	values := make([]uint16, points)
	for index := range values {
		values[index] = binary.BigEndian.Uint16(data[index*2:])
	}
	return values, nil
}

// DecodeStrictMemoryBits validates exact bit response values and cardinality.
func DecodeStrictMemoryBits(data []byte, selector StrictMemorySelector, points int) ([]bool, error) {
	if selector.Bit < 0 {
		return nil, errors.New("FINS bit response requires a bit selector")
	}
	if err := validateStrictMemoryRange(selector, points); err != nil {
		return nil, err
	}
	if len(data) != points {
		return nil, errors.New("FINS memory response cardinality mismatch")
	}
	values := make([]bool, points)
	for index, value := range data {
		if value > 1 {
			return nil, errors.New("FINS bit response contains an invalid value")
		}
		values[index] = value == 1
	}
	return values, nil
}

func strictMemoryAreaCode(area string, bank int, bit bool) (byte, error) {
	wordCodes := map[string]byte{"CIO": 0xb0, "WR": 0xb1, "HR": 0xb2, "AR": 0xb3, "DM": 0x82}
	bitCodes := map[string]byte{"CIO": 0x30, "WR": 0x31, "HR": 0x32, "AR": 0x33, "DM": 0x02}
	if area != "EM" {
		if bit {
			code, exists := bitCodes[area]
			if !exists {
				return 0, errors.New("unknown FINS memory area")
			}
			return code, nil
		}
		code, exists := wordCodes[area]
		if !exists {
			return 0, errors.New("unknown FINS memory area")
		}
		return code, nil
	}
	if bank < 0 || bank > 0x18 {
		return 0, errors.New("invalid FINS EM bank")
	}
	if bank <= 0x0f {
		if bit {
			return byte(0x20 + bank), nil
		}
		return byte(0xa0 + bank), nil
	}
	if bit {
		return byte(0xe0 + bank - 0x10), nil
	}
	return byte(0x60 + bank - 0x10), nil
}

func validateStrictMemoryRange(selector StrictMemorySelector, points int) error {
	code, err := strictMemoryAreaCode(selector.Area, selector.Bank, selector.Bit >= 0)
	if err != nil || code != selector.Code || points < 1 || points > 0xffff {
		return errors.New("invalid FINS memory selector or point count")
	}
	if selector.Bit < 0 {
		if uint64(selector.Word)+uint64(points) > 0x10000 {
			return errors.New("FINS word range exceeds memory address space")
		}
	} else if uint64(selector.Word)*16+uint64(selector.Bit)+uint64(points) > 0x100000 {
		return errors.New("FINS bit range exceeds memory address space")
	}
	return nil
}

func strictMemoryAddressAndCount(selector StrictMemorySelector, points int) []byte {
	bit := byte(0)
	if selector.Bit >= 0 {
		bit = byte(selector.Bit)
	}
	return []byte{selector.Code, byte(selector.Word >> 8), byte(selector.Word), bit, byte(points >> 8), byte(points)}
}
