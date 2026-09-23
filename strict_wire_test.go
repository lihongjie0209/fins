package fins

import (
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

func TestStrictFINSFrameRoundTripAndCorrelation(t *testing.T) {
	header := StrictHeader{
		GatewayCount: 2,
		Destination:  StrictAddress{Network: 1, Node: 10, Unit: 3},
		Source:       StrictAddress{Network: 2, Node: 20, Unit: 4},
		SID:          7,
	}
	command := StrictCommand{Main: 1, Sub: 1, Data: []byte{0x82, 0, 100, 0, 0, 2}}
	wire, err := EncodeStrictRequest(header, command, 64)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x80, 0, 2, 1, 10, 3, 2, 20, 4, 7, 1, 1, 0x82, 0, 100, 0, 0, 2}
	if !reflect.DeepEqual(wire, want) {
		t.Fatalf("wire=%x want=%x", wire, want)
	}
	responseWire := []byte{0xc0, 0, 2, 2, 20, 4, 1, 10, 3, 7, 1, 1, 0, 0, 0x12, 0x34}
	response, err := DecodeStrictResponse(responseWire, header, command, 64)
	if err != nil || response.EndCode != 0 || !reflect.DeepEqual(response.Data, []byte{0x12, 0x34}) {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	response.Data[0] = 0xff
	again, err := DecodeStrictResponse(responseWire, header, command, 64)
	if err != nil || again.Data[0] != 0x12 {
		t.Fatal("response data is not owned")
	}
	for index := 2; index <= 11; index++ {
		bad := append([]byte(nil), responseWire...)
		bad[index]++
		if _, err := DecodeStrictResponse(bad, header, command, 64); err == nil {
			t.Fatalf("accepted mismatched field %d", index)
		}
	}
}

func TestStrictFINSTCPEnvelopeAndNodeHandshake(t *testing.T) {
	request, err := EncodeStrictTCPPacket(0, 0, []byte{0, 0, 0, 20}, 64)
	if err != nil {
		t.Fatal(err)
	}
	if string(request[:4]) != "FINS" || binary.BigEndian.Uint32(request[4:]) != 12 || binary.BigEndian.Uint32(request[8:]) != 0 {
		t.Fatalf("request=%x", request)
	}
	packet, err := DecodeStrictTCPPacket(request, 0, 64)
	if err != nil || !reflect.DeepEqual(packet.Data, []byte{0, 0, 0, 20}) {
		t.Fatalf("packet=%#v err=%v", packet, err)
	}
	response, _ := EncodeStrictTCPPacket(1, 0, []byte{0, 0, 0, 20, 0, 0, 0, 10}, 64)
	client, server, err := DecodeStrictNodeAddressResponse(response, 20, 10, 64)
	if err != nil || client != 20 || server != 10 {
		t.Fatalf("client=%d server=%d err=%v", client, server, err)
	}
	if _, _, err := DecodeStrictNodeAddressResponse(response, 21, 10, 64); err == nil {
		t.Fatal("accepted mismatched configured client node")
	}
	failed, _ := EncodeStrictTCPPacket(1, 0x22, nil, 64)
	if _, _, err := DecodeStrictNodeAddressResponse(failed, 0, 10, 64); !errors.Is(err, ErrStrictTCPStatus) {
		t.Fatalf("err=%v", err)
	}
}

func TestStrictFINSWireRejectsMalformedAndBounds(t *testing.T) {
	header := StrictHeader{Destination: StrictAddress{Node: 10}, Source: StrictAddress{Node: 20}}
	if _, err := EncodeStrictRequest(header, StrictCommand{}, 64); err == nil {
		t.Fatal("accepted empty command")
	}
	if _, err := EncodeStrictRequest(header, StrictCommand{Main: 1, Data: make([]byte, 64)}, 64); err == nil {
		t.Fatal("accepted oversized request")
	}
	if _, err := DecodeStrictResponse([]byte{0xc0}, header, StrictCommand{Main: 1}, 64); err == nil {
		t.Fatal("accepted short response")
	}
	packet, _ := EncodeStrictTCPPacket(3, 0, []byte{1, 2}, 64)
	packet[4+3]++
	if _, err := DecodeStrictTCPPacket(packet, 3, 64); err == nil {
		t.Fatal("accepted invalid declared length")
	}
	packet, _ = EncodeStrictTCPPacket(3, 1, nil, 64)
	if _, err := DecodeStrictTCPPacket(packet, 3, 64); !errors.Is(err, ErrStrictTCPStatus) {
		t.Fatalf("err=%v", err)
	}
}
