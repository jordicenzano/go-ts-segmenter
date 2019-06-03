package tspacket

import (
	"bytes"
	"encoding/binary"
	"io"
)

const (
	// TsDefaultPacketSize Default TS packet size
	TsDefaultPacketSize = 188

	// TsStartByte Start byte for TS pakcets
	TsStartByte = 0x47
)

type transportPacket struct {
	SyncByte                      uint8
	ErrorIndicatorPayloadUnitPid  uint16
	ScrambledAdapFieldContCounter uint8
}

type transportPacketAdaptationField struct {
	Length        uint8
	PresenceFlags uint8
}

// TsPacket Transport stream packet
type TsPacket struct {
	buf       []byte
	lastIndex int
}

// PidTable indicated the first video and audio PIDs found
type PidTable struct {
	firstVideoPid int32
	firstAudioPid int32
}

type transportPacketPCR struct {
	PcrFisrt32 int32
	Pcr16      int16
}

// New Creates a TsPacket instance
func New(packetSize int) TsPacket {
	p := TsPacket{make([]byte, packetSize), 0}

	return p
}

// Reset packet
func (p *TsPacket) Reset() {
	p.lastIndex = 0
}

// AddData Adds bytes to the packet
func (p *TsPacket) AddData(buf []byte) {

	p.lastIndex = p.lastIndex + copy(p.buf[p.lastIndex:], buf[:])
}

// IsComplete Adds bytes to the packet
func (p *TsPacket) IsComplete() bool {
	if p.lastIndex == TsDefaultPacketSize && p.buf[0] == TsStartByte {
		return true
	}
	return false
}

func (p *TsPacket) getAdaptationFieldData(r io.Reader, transportPacketData transportPacket) *transportPacketAdaptationField {
	// Detects if there is adaptation field
	adaptationField := int((transportPacketData.ScrambledAdapFieldContCounter & 0x30) >> 4)
	if adaptationField != 2 && adaptationField != 3 {
		return nil
	}

	var transportPacketAdaptationFieldData transportPacketAdaptationField
	err := binary.Read(r, binary.BigEndian, &transportPacketAdaptationFieldData)
	if err != nil {
		return nil
	}

	return &transportPacketAdaptationFieldData
}

func (p *TsPacket) getPCRS(r io.Reader, transportPacketAdaptationFieldData transportPacketAdaptationField) *float64 {

	if transportPacketAdaptationFieldData.PresenceFlags&0x00 == 0x10 {
		return nil
	}

	var transportPacketPCRData transportPacketPCR
	err := binary.Read(r, binary.BigEndian, &transportPacketPCRData)
	if err != nil {
		return nil
	}

	pcrBase := int64(transportPacketPCRData.PcrFisrt32)*2 + (int64(transportPacketPCRData.Pcr16))&0x1
	pcrExtension := int64(transportPacketPCRData.Pcr16 & 0x1FF)

	var pcrS float64 = -1
	if pcrExtension > 0 {
		pcrS = float64(pcrBase*300.0+pcrExtension) / (27.0 * 1000000.0)
	} else {
		pcrS = float64(pcrBase) / 90000.0
	}

	return &pcrS
}

// GetPCRS Adds bytes to the packet
func (p *TsPacket) GetPCRS() (pcrS float64) {
	pcrS = -1

	if !p.IsComplete() {
		return
	}

	var transportPacketData transportPacket
	r := bytes.NewReader(p.buf)
	err := binary.Read(r, binary.BigEndian, &transportPacketData)
	if err != nil {
		return
	}

	transportPacketAdaptationFieldData := p.getAdaptationFieldData(r, transportPacketData)
	if transportPacketAdaptationFieldData == nil {
		return
	}

	pcrSret := p.getPCRS(r, *transportPacketAdaptationFieldData)
	if pcrSret == nil {
		return
	}

	return *pcrSret
}

// GetPID Adds bytes to the packet
func (p *TsPacket) GetPID() (pID int) {
	pID = -1

	if !p.IsComplete() {
		return
	}

	var transportPacketData transportPacket
	r := bytes.NewReader(p.buf)
	err := binary.Read(r, binary.BigEndian, &transportPacketData)
	if err != nil {
		return
	}

	pID = int(transportPacketData.ErrorIndicatorPayloadUnitPid & 0x1FFF)

	return
}

// GetIDR Adds bytes to the packet
func (p *TsPacket) GetIDR() (isIDR bool) {
	isIDR = false

	if !p.IsComplete() {
		return
	}

	var transportPacketData transportPacket
	r := bytes.NewReader(p.buf)
	err := binary.Read(r, binary.BigEndian, &transportPacketData)
	if err != nil {
		return
	}

	transportPacketAdaptationFieldData := p.getAdaptationFieldData(r, transportPacketData)
	if transportPacketAdaptationFieldData == nil {
		return
	}

	idrFlag := transportPacketAdaptationFieldData.PresenceFlags & 0x40
	if idrFlag > 0 {
		isIDR = true
	}

	return
}
