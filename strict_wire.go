package fins

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ErrStrictTCPStatus marks a nonzero FINS/TCP envelope status.
var ErrStrictTCPStatus = errors.New("FINS/TCP status")

// StrictAddress is a FINS network, node and unit triple.
type StrictAddress struct {
	Network uint8
	Node    uint8
	Unit    uint8
}

// StrictHeader identifies one correlated FINS command.
type StrictHeader struct {
	GatewayCount uint8
	Destination  StrictAddress
	Source       StrictAddress
	SID          uint8
}

// StrictCommand contains a two-byte FINS command code and its payload.
type StrictCommand struct {
	Main uint8
	Sub  uint8
	Data []byte
}

// StrictResponse contains the FINS end code and owned response payload.
type StrictResponse struct {
	EndCode uint16
	Data    []byte
}

// StrictTCPPacket contains one validated FINS/TCP envelope payload.
type StrictTCPPacket struct {
	Command uint32
	Data    []byte
}

// EncodeStrictRequest encodes one bounded FINS command frame.
func EncodeStrictRequest(header StrictHeader, command StrictCommand, maximumFrame int) ([]byte, error) {
	length := 12 + len(command.Data)
	if header.GatewayCount > 7 || !strictNode(header.Destination.Node) || !strictNode(header.Source.Node) || command.Main == 0 || maximumFrame < 12 || length > maximumFrame {
		return nil, errors.New("invalid FINS request or frame limit")
	}
	wire := make([]byte, length)
	wire[0], wire[1], wire[2] = 0x80, 0, header.GatewayCount
	wire[3], wire[4], wire[5] = header.Destination.Network, header.Destination.Node, header.Destination.Unit
	wire[6], wire[7], wire[8] = header.Source.Network, header.Source.Node, header.Source.Unit
	wire[9], wire[10], wire[11] = header.SID, command.Main, command.Sub
	copy(wire[12:], command.Data)
	return wire, nil
}

// DecodeStrictResponse validates response correlation and returns owned data.
func DecodeStrictResponse(wire []byte, request StrictHeader, command StrictCommand, maximumFrame int) (StrictResponse, error) {
	if maximumFrame < 14 || len(wire) < 14 || len(wire) > maximumFrame {
		return StrictResponse{}, errors.New("invalid FINS response size")
	}
	if wire[0] != 0xc0 || wire[1] != 0 || wire[2] != request.GatewayCount {
		return StrictResponse{}, errors.New("FINS response control fields mismatch")
	}
	if wire[3] != request.Source.Network || wire[4] != request.Source.Node || wire[5] != request.Source.Unit ||
		wire[6] != request.Destination.Network || wire[7] != request.Destination.Node || wire[8] != request.Destination.Unit {
		return StrictResponse{}, errors.New("FINS response addresses mismatch")
	}
	if wire[9] != request.SID || wire[10] != command.Main || wire[11] != command.Sub {
		return StrictResponse{}, errors.New("FINS response SID or command mismatch")
	}
	return StrictResponse{
		EndCode: binary.BigEndian.Uint16(wire[12:14]),
		Data:    append([]byte(nil), wire[14:]...),
	}, nil
}

// EncodeStrictTCPPacket encodes one bounded FINS/TCP envelope.
func EncodeStrictTCPPacket(command, status uint32, data []byte, maximumFrame int) ([]byte, error) {
	length := 16 + len(data)
	if maximumFrame < 16 || length > maximumFrame || uint64(len(data)) > uint64(^uint32(0))-8 {
		return nil, errors.New("invalid FINS/TCP packet or frame limit")
	}
	wire := make([]byte, length)
	copy(wire, "FINS")
	binary.BigEndian.PutUint32(wire[4:8], uint32(8+len(data)))
	binary.BigEndian.PutUint32(wire[8:12], command)
	binary.BigEndian.PutUint32(wire[12:16], status)
	copy(wire[16:], data)
	return wire, nil
}

// DecodeStrictTCPPacket validates one exact bounded FINS/TCP envelope.
func DecodeStrictTCPPacket(wire []byte, expectedCommand uint32, maximumFrame int) (StrictTCPPacket, error) {
	if maximumFrame < 16 || len(wire) < 16 || len(wire) > maximumFrame || string(wire[:4]) != "FINS" {
		return StrictTCPPacket{}, errors.New("invalid FINS/TCP packet header or size")
	}
	declared := binary.BigEndian.Uint32(wire[4:8])
	if declared < 8 || uint64(declared)+8 != uint64(len(wire)) {
		return StrictTCPPacket{}, errors.New("FINS/TCP packet length mismatch")
	}
	command := binary.BigEndian.Uint32(wire[8:12])
	if command != expectedCommand {
		return StrictTCPPacket{}, fmt.Errorf("FINS/TCP command %d does not match expected %d", command, expectedCommand)
	}
	if status := binary.BigEndian.Uint32(wire[12:16]); status != 0 {
		return StrictTCPPacket{}, fmt.Errorf("%w 0x%08x", ErrStrictTCPStatus, status)
	}
	return StrictTCPPacket{Command: command, Data: append([]byte(nil), wire[16:]...)}, nil
}

// EncodeStrictNodeAddressRequest builds the FINS/TCP node-address handshake.
func EncodeStrictNodeAddressRequest(sourceNode uint8, maximumFrame int) ([]byte, error) {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(sourceNode))
	return EncodeStrictTCPPacket(0, 0, data, maximumFrame)
}

// DecodeStrictNodeAddressResponse validates assigned client/server nodes.
func DecodeStrictNodeAddressResponse(wire []byte, configuredSource, configuredDestination uint8, maximumFrame int) (uint8, uint8, error) {
	packet, err := DecodeStrictTCPPacket(wire, 1, maximumFrame)
	if err != nil {
		return 0, 0, err
	}
	if len(packet.Data) != 8 {
		return 0, 0, errors.New("FINS/TCP node address response length mismatch")
	}
	client := binary.BigEndian.Uint32(packet.Data[:4])
	server := binary.BigEndian.Uint32(packet.Data[4:])
	if client == 0 || client > 254 || server == 0 || server > 254 {
		return 0, 0, errors.New("FINS/TCP node address response contains invalid nodes")
	}
	if configuredSource != 0 && client != uint32(configuredSource) {
		return 0, 0, errors.New("FINS/TCP assigned client node mismatch")
	}
	if configuredDestination != 0 && server != uint32(configuredDestination) {
		return 0, 0, errors.New("FINS/TCP server node mismatch")
	}
	return uint8(client), uint8(server), nil
}

func strictNode(node uint8) bool { return node != 0 && node != 0xff }
