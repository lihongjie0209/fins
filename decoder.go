package fins

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrInvalidAddress      = errors.New("invalid address format")
	ErrUnsupportedArea     = errors.New("unsupported memory area")
	ErrUnsupportedDataType = errors.New("unsupported data type")
	ErrInvalidDataType     = errors.New("invalid data type")
	ErrDataTooShort        = errors.New("data too short for data type")
)

type Decoder struct {
	areaMap map[string]AreaInfo
}

func NewDecoder() *Decoder {
	d := &Decoder{
		areaMap: make(map[string]AreaInfo),
	}
	d.initAreaMap()
	return d
}

func (d *Decoder) initAreaMap() {
	d.areaMap["CIO"] = AreaInfo{Code: MemoryAreaCIOWord, Name: "CIO", ReadOnly: false, SupportBit: true, SupportString: true}
	d.areaMap["W"] = AreaInfo{Code: MemoryAreaWRWord, Name: "Work", ReadOnly: false, SupportBit: true, SupportString: true}
	d.areaMap["H"] = AreaInfo{Code: MemoryAreaHRWord, Name: "Holding", ReadOnly: false, SupportBit: true, SupportString: true}
	d.areaMap["A"] = AreaInfo{Code: MemoryAreaARWord, Name: "Auxiliary", ReadOnly: true, SupportBit: true, SupportString: true}
	d.areaMap["D"] = AreaInfo{Code: MemoryAreaDMWord, Name: "Data Memory", ReadOnly: false, SupportBit: true, SupportString: true}
	d.areaMap["P"] = AreaInfo{Code: MemoryAreaPVWord, Name: "PVs", ReadOnly: false, SupportBit: true, SupportString: false}
	d.areaMap["F"] = AreaInfo{Code: MemoryAreaFlagBit, Name: "Flag", ReadOnly: true, SupportBit: true, SupportString: false}
	d.areaMap["T"] = AreaInfo{Code: MemoryAreaTimerPV, Name: "Timer", ReadOnly: false, SupportBit: true, SupportString: false}
	d.areaMap["C"] = AreaInfo{Code: MemoryAreaCounterPV, Name: "Counter", ReadOnly: false, SupportBit: true, SupportString: false}

	for i := 0; i <= 15; i++ {
		key := fmt.Sprintf("EM%d", i)
		wordCode, _ := EMWordArea(i)
		d.areaMap[key] = AreaInfo{Code: wordCode, Name: fmt.Sprintf("EM%d", i), ReadOnly: false, SupportBit: true, SupportString: true}
	}
}

func (d *Decoder) GetAreaInfo(areaName string) (AreaInfo, bool) {
	info, ok := d.areaMap[strings.ToUpper(areaName)]
	return info, ok
}

func (d *Decoder) GetAreaCode(areaName string) (uint8, error) {
	info, ok := d.GetAreaInfo(areaName)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedArea, areaName)
	}
	return info.Code, nil
}

var addressRegex = regexp.MustCompile(`^(CIO|EM\d+W?|[WHDAPTC])(\d+)(?:\.(\d+)([HL])?)?$`)

func (d *Decoder) ParseAddress(addr string) (*ParsedAddress, error) {
	addr = strings.ToUpper(strings.TrimSpace(addr))

	matches := addressRegex.FindStringSubmatch(addr)
	if matches == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidAddress, addr)
	}

	areaName := matches[1]
	addrStr := matches[2]
	dotNumStr := matches[3]
	byteOrderStr := matches[4]

	lookupArea := areaName
	if strings.HasPrefix(areaName, "EM") && strings.HasSuffix(areaName, "W") {
		lookupArea = areaName[:len(areaName)-1]
	}

	areaInfo, ok := d.GetAreaInfo(lookupArea)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedArea, areaName)
	}

	address, err := strconv.Atoi(addrStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid address number %s", ErrInvalidAddress, addrStr)
	}

	parsed := &ParsedAddress{
		Area:     lookupArea,
		AreaCode: areaInfo.Code,
		Address:  address,
		Bit:      -1,
	}

	if strings.HasPrefix(lookupArea, "EM") {
		emNum, _ := strconv.Atoi(strings.TrimPrefix(lookupArea, "EM"))
		parsed.EMNumber = emNum
	}

	if dotNumStr != "" {
		dotNum, err := strconv.Atoi(dotNumStr)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid number %s after dot", ErrInvalidAddress, dotNumStr)
		}

		isString := byteOrderStr != "" || (lookupArea == "D" && dotNum > 15)
		isBit := !isString && dotNum >= 0 && dotNum <= 15

		if isString {
			if !areaInfo.SupportString {
				return nil, fmt.Errorf("%w: area %s does not support string access", ErrInvalidAddress, lookupArea)
			}
			if dotNum <= 0 {
				return nil, fmt.Errorf("%w: string length must be positive", ErrInvalidAddress)
			}
			parsed.StringLen = dotNum
			parsed.IsString = true
			parsed.ByteOrder = ByteOrderLow
			if byteOrderStr == "H" {
				parsed.ByteOrder = ByteOrderHigh
			}
		} else if isBit {
			if !areaInfo.SupportBit {
				return nil, fmt.Errorf("%w: area %s does not support bit access", ErrInvalidAddress, lookupArea)
			}
			parsed.Bit = dotNum
			parsed.IsBit = true
		} else {
			return nil, fmt.Errorf("%w: invalid address format: %s", ErrInvalidAddress, addr)
		}
	}

	return parsed, nil
}

func (d *Decoder) EncodeValue(value interface{}, dataType DataType, addr *ParsedAddress) ([]byte, error) {
	if addr != nil && addr.IsBit {
		boolVal, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("%w: expected bool for bit access", ErrInvalidDataType)
		}
		if boolVal {
			return []byte{0x01}, nil
		}
		return []byte{0x00}, nil
	}

	var bo binary.ByteOrder = binary.BigEndian
	if addr != nil && addr.IsString && addr.ByteOrder == ByteOrderLow {
		bo = binary.LittleEndian
	}

	switch dataType {
	case DataTypeUINT8:
		v, ok := value.(uint8)
		if !ok {
			if f, ok := value.(float64); ok {
				v = uint8(f)
			} else {
				return nil, fmt.Errorf("%w: expected uint8", ErrInvalidDataType)
			}
		}
		return []byte{v}, nil

	case DataTypeINT8:
		v, ok := value.(int8)
		if !ok {
			if f, ok := value.(float64); ok {
				v = int8(f)
			} else {
				return nil, fmt.Errorf("%w: expected int8", ErrInvalidDataType)
			}
		}
		return []byte{uint8(v)}, nil

	case DataTypeUINT16:
		v, ok := value.(uint16)
		if !ok {
			if f, ok := value.(float64); ok {
				v = uint16(f)
			} else {
				return nil, fmt.Errorf("%w: expected uint16", ErrInvalidDataType)
			}
		}
		buf := make([]byte, 2)
		bo.PutUint16(buf, v)
		return buf, nil

	case DataTypeINT16:
		v, ok := value.(int16)
		if !ok {
			if f, ok := value.(float64); ok {
				v = int16(f)
			} else {
				return nil, fmt.Errorf("%w: expected int16", ErrInvalidDataType)
			}
		}
		buf := make([]byte, 2)
		bo.PutUint16(buf, uint16(v))
		return buf, nil

	case DataTypeUINT32:
		v, ok := value.(uint32)
		if !ok {
			if f, ok := value.(float64); ok {
				v = uint32(f)
			} else {
				return nil, fmt.Errorf("%w: expected uint32", ErrInvalidDataType)
			}
		}
		buf := make([]byte, 4)
		bo.PutUint32(buf, v)
		return buf, nil

	case DataTypeINT32:
		v, ok := value.(int32)
		if !ok {
			if f, ok := value.(float64); ok {
				v = int32(f)
			} else {
				return nil, fmt.Errorf("%w: expected int32", ErrInvalidDataType)
			}
		}
		buf := make([]byte, 4)
		bo.PutUint32(buf, uint32(v))
		return buf, nil

	case DataTypeUINT64:
		v, ok := value.(uint64)
		if !ok {
			if f, ok := value.(float64); ok {
				v = uint64(f)
			} else {
				return nil, fmt.Errorf("%w: expected uint64", ErrInvalidDataType)
			}
		}
		buf := make([]byte, 8)
		bo.PutUint64(buf, v)
		return buf, nil

	case DataTypeINT64:
		v, ok := value.(int64)
		if !ok {
			if f, ok := value.(float64); ok {
				v = int64(f)
			} else {
				return nil, fmt.Errorf("%w: expected int64", ErrInvalidDataType)
			}
		}
		buf := make([]byte, 8)
		bo.PutUint64(buf, uint64(v))
		return buf, nil

	case DataTypeFLOAT:
		v, ok := value.(float32)
		if !ok {
			if f, ok := value.(float64); ok {
				v = float32(f)
			} else {
				return nil, fmt.Errorf("%w: expected float32", ErrInvalidDataType)
			}
		}
		buf := make([]byte, 4)
		bo.PutUint32(buf, math.Float32bits(v))
		return buf, nil

	case DataTypeDOUBLE:
		v, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("%w: expected float64", ErrInvalidDataType)
		}
		buf := make([]byte, 8)
		bo.PutUint64(buf, math.Float64bits(v))
		return buf, nil

	case DataTypeSTRING:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%w: expected string", ErrInvalidDataType)
		}
		strLen := len(s)
		if addr != nil && addr.StringLen > 0 {
			strLen = addr.StringLen
		}
		buf := make([]byte, strLen)
		copy(buf, s)
		return buf, nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDataType, dataType)
	}
}

func (d *Decoder) DecodeValue(data []byte, dataType DataType, addr *ParsedAddress) (interface{}, error) {
	if addr != nil && addr.IsBit {
		if len(data) < 1 {
			return nil, ErrDataTooShort
		}
		return data[0]&0x01 != 0, nil
	}

	var bo binary.ByteOrder = binary.BigEndian
	if addr != nil && addr.IsString && addr.ByteOrder == ByteOrderLow {
		bo = binary.LittleEndian
	}

	switch dataType {
	case DataTypeUINT8:
		if len(data) < 1 {
			return nil, ErrDataTooShort
		}
		return data[0], nil

	case DataTypeINT8:
		if len(data) < 1 {
			return nil, ErrDataTooShort
		}
		return int8(data[0]), nil

	case DataTypeUINT16:
		if len(data) < 2 {
			return nil, ErrDataTooShort
		}
		return bo.Uint16(data[:2]), nil

	case DataTypeINT16:
		if len(data) < 2 {
			return nil, ErrDataTooShort
		}
		return int16(bo.Uint16(data[:2])), nil

	case DataTypeUINT32:
		if len(data) < 4 {
			return nil, ErrDataTooShort
		}
		return bo.Uint32(data[:4]), nil

	case DataTypeINT32:
		if len(data) < 4 {
			return nil, ErrDataTooShort
		}
		return int32(bo.Uint32(data[:4])), nil

	case DataTypeUINT64:
		if len(data) < 8 {
			return nil, ErrDataTooShort
		}
		return bo.Uint64(data[:8]), nil

	case DataTypeINT64:
		if len(data) < 8 {
			return nil, ErrDataTooShort
		}
		return int64(bo.Uint64(data[:8])), nil

	case DataTypeFLOAT:
		if len(data) < 4 {
			return nil, ErrDataTooShort
		}
		return math.Float32frombits(bo.Uint32(data[:4])), nil

	case DataTypeDOUBLE:
		if len(data) < 8 {
			return nil, ErrDataTooShort
		}
		return math.Float64frombits(bo.Uint64(data[:8])), nil

	case DataTypeSTRING:
		strLen := len(data)
		if addr != nil && addr.StringLen > 0 && addr.StringLen < strLen {
			strLen = addr.StringLen
		}
		s := string(data[:strLen])
		if idx := strings.IndexByte(s, 0); idx >= 0 {
			s = s[:idx]
		}
		return s, nil

	case DataTypeBIT:
		if len(data) < 1 {
			return nil, ErrDataTooShort
		}
		return data[0]&0x01 != 0, nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDataType, dataType)
	}
}

func (d *Decoder) DataTypeSize(dataType DataType) (int, error) {
	switch dataType {
	case DataTypeUINT8, DataTypeINT8:
		return 1, nil
	case DataTypeUINT16, DataTypeINT16:
		return 2, nil
	case DataTypeUINT32, DataTypeINT32, DataTypeFLOAT:
		return 4, nil
	case DataTypeUINT64, DataTypeINT64, DataTypeDOUBLE:
		return 8, nil
	case DataTypeSTRING:
		return 0, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnsupportedDataType, dataType)
	}
}

func (d *Decoder) IsBitDataType(dataType DataType) bool {
	return dataType == DataTypeBIT
}

func GetWordAreaCode(bitAreaCode uint8) (uint8, error) {
	switch bitAreaCode {
	case MemoryAreaCIOBit:
		return MemoryAreaCIOWord, nil
	case MemoryAreaWRBit:
		return MemoryAreaWRWord, nil
	case MemoryAreaHRBit:
		return MemoryAreaHRWord, nil
	case MemoryAreaARBit:
		return MemoryAreaARWord, nil
	case MemoryAreaDMBit:
		return MemoryAreaDMWord, nil
	case MemoryAreaPVBit:
		return MemoryAreaPVWord, nil
	case MemoryAreaFlagBit:
		return MemoryAreaFlagBit, nil
	default:
		if bitAreaCode >= MemoryAreaEM0Bit && bitAreaCode <= MemoryAreaEM15Bit {
			return MemoryAreaEM0Word + (bitAreaCode - MemoryAreaEM0Bit), nil
		}
		return 0, fmt.Errorf("unknown bit area code: 0x%02x", bitAreaCode)
	}
}

func GetBitAreaCode(wordAreaCode uint8) (uint8, error) {
	switch wordAreaCode {
	case MemoryAreaCIOWord:
		return MemoryAreaCIOBit, nil
	case MemoryAreaWRWord:
		return MemoryAreaWRBit, nil
	case MemoryAreaHRWord:
		return MemoryAreaHRBit, nil
	case MemoryAreaARWord:
		return MemoryAreaARBit, nil
	case MemoryAreaDMWord:
		return MemoryAreaDMBit, nil
	case MemoryAreaPVWord:
		return MemoryAreaPVBit, nil
	default:
		if wordAreaCode >= MemoryAreaEM0Word && wordAreaCode <= MemoryAreaEM15Word {
			return MemoryAreaEM0Bit + (wordAreaCode - MemoryAreaEM0Word), nil
		}
		return 0, fmt.Errorf("unknown word area code: 0x%02x", wordAreaCode)
	}
}
