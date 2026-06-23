package fins

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Scheduler struct {
	transport *Transport
	decoder   *Decoder

	maxFrameLength int
	minInterval    time.Duration

	totalRequests atomic.Int64
	successCount  atomic.Int64
	failureCount  atomic.Int64

	lastRequestTime time.Time
	mu              sync.Mutex
}

func NewScheduler(transport *Transport, decoder *Decoder, cfg map[string]interface{}) *Scheduler {
	s := &Scheduler{
		transport:      transport,
		decoder:        decoder,
		maxFrameLength: 64,
		minInterval:    0,
	}

	if v, ok := cfg["maxFrameLength"].(float64); ok {
		s.maxFrameLength = int(v)
	} else if v, ok := cfg["maxFrameLength"].(int); ok {
		s.maxFrameLength = v
	}

	if v, ok := cfg["minInterval"].(float64); ok {
		s.minInterval = time.Duration(v) * time.Millisecond
	} else if v, ok := cfg["minInterval"].(int); ok {
		s.minInterval = time.Duration(v) * time.Millisecond
	}

	return s
}

type pointGroup struct {
	areaCode  uint8
	startAddr uint16
	count     uint16
	points    []groupedPoint
	isBit     bool
}

type groupedPoint struct {
	point  Point
	parsed *ParsedAddress
	offset int
}

func (s *Scheduler) ReadPoints(ctx context.Context, points []Point) (map[string]Value, error) {
	if len(points) == 0 {
		return make(map[string]Value), nil
	}

	results := make(map[string]Value)

	parsedPoints := make([]struct {
		point  Point
		parsed *ParsedAddress
		err    error
	}, len(points))

	for i, p := range points {
		parsed, err := s.decoder.ParseAddress(p.Address)
		parsedPoints[i] = struct {
			point  Point
			parsed *ParsedAddress
			err    error
		}{point: p, parsed: parsed, err: err}

		if err != nil {
			results[p.ID] = Value{
				Value:   nil,
				Quality: QualityBad,
				TS:      time.Now(),
			}
		}
	}

	groups := s.groupPoints(parsedPoints)

	for _, group := range groups {
		s.waitMinInterval()

		s.totalRequests.Add(1)

		var data []byte
		var err error

		if group.isBit {
			data, err = s.readBitGroup(ctx, group)
		} else {
			data, err = s.readWordGroup(ctx, group)
		}

		if err != nil {
			s.failureCount.Add(1)
			for _, gp := range group.points {
				results[gp.point.ID] = Value{
					Value:   nil,
					Quality: QualityBad,
					TS:      time.Now(),
				}
			}
			continue
		}

		s.successCount.Add(1)

		now := time.Now()
		for _, gp := range group.points {
			val, err := s.decodePointValue(data, gp, group)
			if err != nil {
				results[gp.point.ID] = Value{
					Value:   nil,
					Quality: QualityBad,
					TS:      now,
				}
			} else {
				results[gp.point.ID] = Value{
					Value:   val,
					Quality: QualityGood,
					TS:      now,
				}
			}
		}
	}

	return results, nil
}

func (s *Scheduler) groupPoints(parsedPoints []struct {
	point  Point
	parsed *ParsedAddress
	err    error
}) []pointGroup {

	type groupKey struct {
		areaCode uint8
		isBit    bool
	}

	groupsMap := make(map[groupKey][]groupedPoint)

	for _, pp := range parsedPoints {
		if pp.err != nil {
			continue
		}

		isBit := pp.parsed.IsBit || pp.point.DataType == DataTypeBIT

		key := groupKey{
			areaCode: pp.parsed.AreaCode,
			isBit:    isBit,
		}

		if isBit {
			bitCode, err := GetBitAreaCode(pp.parsed.AreaCode)
			if err == nil {
				key.areaCode = bitCode
			}
		}

		gp := groupedPoint{
			point:  pp.point,
			parsed: pp.parsed,
		}

		groupsMap[key] = append(groupsMap[key], gp)
	}

	var result []pointGroup

	for key, points := range groupsMap {
		sort.Slice(points, func(i, j int) bool {
			if points[i].parsed.Address != points[j].parsed.Address {
				return points[i].parsed.Address < points[j].parsed.Address
			}
			return points[i].parsed.Bit < points[j].parsed.Bit
		})

		if key.isBit {
			for _, p := range points {
				group := pointGroup{
					areaCode:  key.areaCode,
					startAddr: uint16(p.parsed.Address),
					count:     1,
					points:    []groupedPoint{p},
					isBit:     true,
				}
				p.offset = 0
				result = append(result, group)
			}
		} else {
			currentGroup := pointGroup{
				areaCode: key.areaCode,
				isBit:    false,
			}

			maxBytes := s.maxFrameLength * 2

			for _, p := range points {
				typeSize, _ := s.decoder.DataTypeSize(p.point.DataType)
				if typeSize == 0 {
					if p.parsed.IsString {
						typeSize = p.parsed.StringLen
					} else {
						typeSize = 2
					}
				}

				wordOffset := p.parsed.Address
				wordCount := (typeSize + 1) / 2

				if len(currentGroup.points) == 0 {
					currentGroup.startAddr = uint16(wordOffset)
					currentGroup.count = uint16(wordCount)
					p.offset = 0
					currentGroup.points = append(currentGroup.points, p)
				} else {
					endAddr := int(currentGroup.startAddr) + int(currentGroup.count)
					newEnd := wordOffset + wordCount
					totalBytes := (newEnd - int(currentGroup.startAddr)) * 2

					if wordOffset <= endAddr+10 && totalBytes <= maxBytes {
						if wordOffset+wordCount > int(currentGroup.startAddr)+int(currentGroup.count) {
							currentGroup.count = uint16(wordOffset + wordCount - int(currentGroup.startAddr))
						}
						p.offset = (wordOffset - int(currentGroup.startAddr)) * 2
						currentGroup.points = append(currentGroup.points, p)
					} else {
						result = append(result, currentGroup)
						currentGroup = pointGroup{
							areaCode:  key.areaCode,
							startAddr: uint16(wordOffset),
							count:     uint16(wordCount),
							points:    []groupedPoint{p},
							isBit:     false,
						}
						p.offset = 0
					}
				}
			}

			if len(currentGroup.points) > 0 {
				result = append(result, currentGroup)
			}
		}
	}

	return result
}

func (s *Scheduler) readWordGroup(ctx context.Context, group pointGroup) ([]byte, error) {
	return s.transport.ReadMemory(ctx, group.areaCode, group.startAddr, group.count)
}

func (s *Scheduler) readBitGroup(ctx context.Context, group pointGroup) ([]byte, error) {
	if len(group.points) == 0 {
		return nil, nil
	}
	p := group.points[0]
	bits, err := s.transport.ReadBits(ctx, group.areaCode, uint16(p.parsed.Address), uint8(p.parsed.Bit), 1)
	if err != nil {
		return nil, err
	}
	data := make([]byte, len(bits))
	for i, b := range bits {
		if b {
			data[i] = 0x01
		}
	}
	return data, nil
}

func (s *Scheduler) decodePointValue(data []byte, gp groupedPoint, group pointGroup) (interface{}, error) {
	if group.isBit {
		if len(data) < 1 {
			return nil, ErrDataTooShort
		}
		return data[0]&0x01 != 0, nil
	}

	offset := gp.offset
	typeSize, _ := s.decoder.DataTypeSize(gp.point.DataType)

	if gp.parsed.IsString {
		end := offset + gp.parsed.StringLen
		if end > len(data) {
			end = len(data)
		}
		strData := data[offset:end]
		return s.decoder.DecodeValue(strData, DataTypeSTRING, gp.parsed)
	}

	if typeSize == 0 {
		typeSize = 2
	}

	if offset+typeSize > len(data) {
		return nil, ErrDataTooShort
	}

	return s.decoder.DecodeValue(data[offset:offset+typeSize], gp.point.DataType, gp.parsed)
}

func (s *Scheduler) WritePoint(ctx context.Context, point Point, value interface{}) error {
	s.totalRequests.Add(1)

	parsed, err := s.decoder.ParseAddress(point.Address)
	if err != nil {
		s.failureCount.Add(1)
		return err
	}

	isBit := parsed.IsBit || point.DataType == DataTypeBIT

	encoded, err := s.decoder.EncodeValue(value, point.DataType, parsed)
	if err != nil {
		s.failureCount.Add(1)
		return err
	}

	s.waitMinInterval()

	if isBit {
		bitArea, err := GetBitAreaCode(parsed.AreaCode)
		if err != nil {
			bitArea = parsed.AreaCode
		}
		boolVal := false
		if len(encoded) > 0 {
			boolVal = encoded[0] != 0
		}
		err = s.transport.WriteBits(ctx, bitArea, uint16(parsed.Address), uint8(parsed.Bit), []bool{boolVal})
		if err != nil {
			s.failureCount.Add(1)
			return err
		}
	} else {
		wordCount := uint16((len(encoded) + 1) / 2)
		if point.DataType == DataTypeSTRING && parsed.StringLen > 0 {
			wordCount = uint16((parsed.StringLen + 1) / 2)
			if len(encoded) < int(wordCount)*2 {
				padded := make([]byte, int(wordCount)*2)
				copy(padded, encoded)
				encoded = padded
			}
		}
		err = s.transport.WriteMemory(ctx, parsed.AreaCode, uint16(parsed.Address), wordCount, encoded)
		if err != nil {
			s.failureCount.Add(1)
			return err
		}
	}

	s.successCount.Add(1)
	return nil
}

func (s *Scheduler) waitMinInterval() {
	if s.minInterval <= 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	elapsed := time.Since(s.lastRequestTime)
	if elapsed < s.minInterval {
		time.Sleep(s.minInterval - elapsed)
	}
	s.lastRequestTime = time.Now()
}

func (s *Scheduler) GetStats() SchedulerStats {
	return SchedulerStats{
		TotalRequests: s.totalRequests.Load(),
		SuccessCount:  s.successCount.Load(),
		FailureCount:  s.failureCount.Load(),
	}
}

func (s *Scheduler) ResetStats() {
	s.totalRequests.Store(0)
	s.successCount.Store(0)
	s.failureCount.Store(0)
}

func (s *Scheduler) Health() HealthStatus {
	if s.transport.IsConnected() {
		return HealthStatusUp
	}
	return HealthStatusDown
}

func (s *Scheduler) MaxFrameLength() int {
	return s.maxFrameLength
}

func (s *Scheduler) SetMaxFrameLength(length int) {
	if length > 0 {
		s.mu.Lock()
		s.maxFrameLength = length
		s.mu.Unlock()
	}
}

func (s *Scheduler) SetMinInterval(interval time.Duration) {
	s.mu.Lock()
	s.minInterval = interval
	s.mu.Unlock()
}

func (s *Scheduler) String() string {
	stats := s.GetStats()
	return fmt.Sprintf("Scheduler{total=%d, success=%d, failure=%d}",
		stats.TotalRequests, stats.SuccessCount, stats.FailureCount)
}
