package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"github.com/kKar1503/rewired-server-2024/internal/utils"
)

const CURRENT_VERSION byte = 1

var (
	ErrRequireVersionAndPacketType = errors.New("require version and packet type byte to be unmarshal first")
	ErrVersionMismatch             = errors.New("version mismatched")
	ErrPacketTypeMismatch          = errors.New("packet type mismatched")
	ErrInvalidPacketType           = errors.New("invalid packet type")
	ErrInvalidGateStatus           = errors.New("invalid gate status")
	ErrEmptyRawData                = errors.New("empty raw data")
	ErrInvalidBinarySize           = errors.New("invalid binary size")
	ErrInvalidTimestamp            = errors.New("invalid timestamp")
)

// The raw TCP packet that is received from the device.
//
// The first byte of the raw packet contains the version number and the packet type with 4 bits each.
//
// With the packet type and the version, we can determine the packet struct to unmarshal the binary to.
type RawPacket struct {
	raw        []byte
	Version    byte
	PacketType PacketType
}

func (p *RawPacket) MarshalBinary() ([]byte, error) {
	if len(p.raw) == 0 {
		return nil, ErrEmptyRawData
	}

	versionAndPacketType := utils.Join2FourBitsIntoByte(p.Version, byte(p.PacketType))

	data := append([]byte{versionAndPacketType}, p.raw...)

	return data, nil
}

func (p *RawPacket) ReadPackets(connReader io.Reader) error {
	firstPacketBuf := make([]byte, 1)

	_, err := connReader.Read(firstPacketBuf)
	if err != nil {
		return err
	}

	firstPacket := firstPacketBuf[0]
	version, packetType := utils.SplitByteInto2FourBits(firstPacket)

	if version != CURRENT_VERSION {
		return ErrVersionMismatch
	}

	p.Version = version

	switch PacketType(packetType) {
	case PacketTypeHeartbeat, PacketTypeDecrement, PacketTypeIncrement:
		p.PacketType = PacketType(packetType)

		packetData := make([]byte, 2)
		_, err := connReader.Read(packetData)
		if err != nil {
			return err
		}

		p.raw = packetData
	case PacketTypeGateStatus:
		p.PacketType = PacketTypeGateStatus

		gateStatusPacketData := make([]byte, 7)
		_, err := connReader.Read(gateStatusPacketData)
		if err != nil {
			return err
		}

		p.raw = gateStatusPacketData
	default:
		return ErrInvalidPacketType
	}

	return nil
}

// The heartbeat packet is received from the device.
//
// The ID is used to identify that the gate is currently still connected to the server,
// giving a live update that the server is up and running.
//
// This packet will be in the size of 4 + 4 + 16 = 24 bits indicating the version of the API + packet type
// + ID of the device.
type HeartbeatPacket struct {
	RawPacket

	GateID uint16
}

func (p *HeartbeatPacket) Parse(rawPacket *RawPacket) error {
	if len(rawPacket.raw) != 2 {
		return ErrInvalidBinarySize
	}

	if rawPacket.PacketType != PacketTypeHeartbeat {
		return ErrPacketTypeMismatch
	}

	p.GateID = binary.BigEndian.Uint16(rawPacket.raw)
	p.Version = rawPacket.Version
	p.PacketType = rawPacket.PacketType

	return nil
}

func (p *HeartbeatPacket) SetGateID(gateID uint16) {
	p.GateID = gateID
	p.raw = make([]byte, 2)
	binary.BigEndian.PutUint16(p.raw, gateID)
}

// The increment packet is received from the device.
//
// The ID is used to increment the population counter that the gate is responsible for, either of the gateID
// can be used for the identification.
//
// This packet will be in the size of 4 + 4 + 16 = 24 bits indicating the version of the API + packet type
// + ID of the device.
type IncrementPacket struct {
	RawPacket

	GateID uint16
}

func (p *IncrementPacket) Parse(rawPacket *RawPacket) error {
	if len(rawPacket.raw) != 2 {
		return ErrInvalidBinarySize
	}

	if rawPacket.PacketType != PacketTypeIncrement {
		return ErrPacketTypeMismatch
	}

	p.GateID = binary.BigEndian.Uint16(rawPacket.raw)
	p.Version = rawPacket.Version
	p.PacketType = rawPacket.PacketType

	return nil
}

func (p *IncrementPacket) SetGateID(gateID uint16) {
	p.GateID = gateID
	p.raw = make([]byte, 2)
	binary.BigEndian.PutUint16(p.raw, gateID)
}

// The decrement packet is received from the device.
//
// The ID is used to decrement the population counter that the gate is responsible for, either of the gateID
// can be used for the identification.
//
// This packet will be in the size of 4 + 4 + 16 = 24 bits indicating the version of the API + packet type
// + ID of the device.
type DecrementPacket struct {
	RawPacket

	GateID uint16
}

func (p *DecrementPacket) Parse(rawPacket *RawPacket) error {
	if len(rawPacket.raw) != 2 {
		return ErrInvalidBinarySize
	}

	if rawPacket.PacketType != PacketTypeDecrement {
		return ErrPacketTypeMismatch
	}

	p.GateID = binary.BigEndian.Uint16(rawPacket.raw)
	p.Version = rawPacket.Version
	p.PacketType = rawPacket.PacketType

	return nil
}

func (p *DecrementPacket) SetGateID(gateID uint16) {
	p.GateID = gateID
	p.raw = make([]byte, 2)
	binary.BigEndian.PutUint16(p.raw, gateID)
}

// The gate status packet is receivedd from the device when there is a change in state.
//
// The status is sent from the device when there is a change in status.
//
// This packet will be in the size of 4 + 4 + 16 + 8 + 32 = 64 bits indicating the version of the API
// + packet type + ID + status + trigger time of the status.
type GateStatusPacket struct {
	RawPacket

	GateID      uint16
	Status      GateStatus
	TriggerTime time.Time
}

func (p *GateStatusPacket) Parse(rawPacket *RawPacket) error {
	if len(rawPacket.raw) != 7 {
		return ErrInvalidBinarySize
	}

	if rawPacket.PacketType != PacketTypeGateStatus {
		return ErrPacketTypeMismatch
	}

	p.GateID = binary.BigEndian.Uint16(rawPacket.raw[:2])
	p.Version = rawPacket.Version
	p.PacketType = rawPacket.PacketType

	var timeData int32
	buf := bytes.NewReader(rawPacket.raw[3:])
	err := binary.Read(buf, binary.BigEndian, &timeData)
	if err != nil {
		return ErrInvalidTimestamp
	}
	p.TriggerTime = time.Unix(int64(timeData), 0)

	switch GateStatus(rawPacket.raw[2]) {
	case GateStatusTurnOn, GateStatusUnblocked, GateStatusBlocked, GateStatusFaulty:
		p.Status = GateStatus(rawPacket.raw[2])
	default:
		return ErrInvalidGateStatus
	}

	return nil
}

func (p *GateStatusPacket) SetGateID(gateID uint16) {
	p.GateID = gateID
	if len(p.raw) == 0 {
		p.raw = make([]byte, 7)
	}

	p.raw[0] = byte(gateID >> 8)
	p.raw[1] = byte(gateID)
}

func (p *GateStatusPacket) SetStatus(status GateStatus) {
	p.Status = status
	if len(p.raw) == 0 {
		p.raw = make([]byte, 7)
	}

	p.raw[2] = byte(status)
}

func (p *GateStatusPacket) SetTimestamp(triggerTime time.Time) error {
	int32Time, err := utils.TimeToInt32(triggerTime)
	if err != nil {
		return err
	}

	p.TriggerTime = triggerTime
	if len(p.raw) == 0 {
		p.raw = make([]byte, 7)
	}

	p.raw[3] = byte(int32Time >> 24)
	p.raw[4] = byte(int32Time >> 16)
	p.raw[5] = byte(int32Time >> 8)
	p.raw[6] = byte(int32Time)

	return nil
}

// The gate status is the enum of all possible status from the device in the GateStatusPacket.
type GateStatus uint8

const (
	GateStatusTurnOn GateStatus = iota + 1
	GateStatusUnblocked
	GateStatusBlocked
	GateStatusFaulty
)

// The packet type is the type of packet that the TCP packet, this is used to determine the packet type
// for unmarshalling purposes.
type PacketType byte

const (
	PacketTypeHeartbeat PacketType = iota + 1
	PacketTypeGateStatus
	PacketTypeIncrement
	PacketTypeDecrement
)
