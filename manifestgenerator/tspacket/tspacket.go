package tspacket

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	// TsDefaultPacketSize Default TS packet size
	TsDefaultPacketSize int = 188

	// MaxPCRSValue (in seconds). 2^33 / 90000 (33 bits used by pcr with timebase of 90KHz)
	MaxPCRSValue float64 = 95443

	// tsStartByte Start byte for TS pakcets
	tsStartByte uint8 = 0x47

	// H264StreamType indicates h264 video ES
	H264StreamType uint8 = 0x1B

	// ADTSStreamType indicates audio ADTS ES
	ADTSStreamType uint8 = 0x0F

	// PATPID PID of PAT table
	PATPID uint16 = 0
)

// transportPacketData TS packet info
type transportPacketData struct {
	valid                      bool
	SyncByte                   uint8
	TransportErrorIndicator    bool
	PayloadUnitStartIndicator  bool
	TransportPriority          bool
	PID                        uint16
	TransportScramblingControl uint8
	AdaptationFieldControl     uint8
	ContinuityCounter          uint8
	AdaptationField            transportPacketAdaptationFieldData
	Pat                        programAddressTable
	Pmt                        programMapTable
}

// Reset transportPacketData
func (t *transportPacketData) Reset() {
	t.valid = false
	t.SyncByte = 0
	t.TransportErrorIndicator = false
	t.PayloadUnitStartIndicator = false
	t.TransportErrorIndicator = false
	t.PID = 0
	t.TransportScramblingControl = 0
	t.AdaptationFieldControl = 0
	t.ContinuityCounter = 0
	t.AdaptationField.AdaptationFieldLength = 0
	t.AdaptationField.DiscontinuityIndicator = false
	t.AdaptationField.RandomAccessIndicator = false
	t.AdaptationField.ElementaryStreamPriorityIndicator = false
	t.AdaptationField.PCRFlag = false
	t.AdaptationField.OPCRFlag = false
	t.AdaptationField.SplicingPointFlag = false
	t.AdaptationField.TransportPrivateDataFlag = false
	t.AdaptationField.AdaptationFieldExtensionFlag = false
	t.AdaptationField.PCRData.valid = false
	t.AdaptationField.PCRData.ProgramClockReferenceBase = 0
	t.AdaptationField.PCRData.ProgramClockReferenceExtension = 0
	t.AdaptationField.PCRData.reserved = 0
	t.AdaptationField.PCRData.PCRs = 0
	t.Pat.valid = false
	t.Pat.PmtPID = 0
	t.Pmt.valid = false
	t.Pmt.AudioADTS = t.Pmt.AudioADTS[:0]
	t.Pmt.Videoh264 = t.Pmt.Videoh264[:0]
	t.Pmt.Other = t.Pmt.Other[:0]
}

// transportPacketAdaptationFieldData TS adaptation field packet info
type transportPacketAdaptationFieldData struct {
	AdaptationFieldLength             uint8
	DiscontinuityIndicator            bool
	RandomAccessIndicator             bool
	ElementaryStreamPriorityIndicator bool
	PCRFlag                           bool
	OPCRFlag                          bool
	SplicingPointFlag                 bool
	TransportPrivateDataFlag          bool
	AdaptationFieldExtensionFlag      bool
	PCRData                           transportPacketAdaptationPCRFieldData
}

// transportPacketAdaptationPCRFieldData TS PCR field packet info
type transportPacketAdaptationPCRFieldData struct {
	ProgramClockReferenceBase      uint64
	reserved                       uint8
	ProgramClockReferenceExtension uint16
	PCRs                           float64
	valid                          bool
}

// PAT data storing the PMT ID
type programAddressTable struct {
	valid  bool
	PmtPID uint16
}

// PMT data storing the video and audio PIDs to process
type programMapTable struct {
	valid     bool
	Videoh264 []uint16
	AudioADTS []uint16
	Other     []uint16
}

// TsPacket Transport stream packet
type TsPacket struct {
	buf             []byte
	lastIndex       int
	transportPacket transportPacketData
	pat             programAddressTable
	pmt             programMapTable
}

// New Creates a TsPacket instance
func New(packetSize int) TsPacket {
	p := TsPacket{make([]byte, packetSize), 0, *new(transportPacketData), programAddressTable{valid: false, PmtPID: 0}, programMapTable{valid: false}}

	return p
}

// CloneFrom Deep clone all the packet
func CloneFrom(srcPckt TsPacket) TsPacket {
	pcktSize := len(srcPckt.buf)

	newPckt := TsPacket{make([]byte, pcktSize), 0, *new(transportPacketData), programAddressTable{valid: false, PmtPID: 0}, programMapTable{valid: false}}
	copy(newPckt.buf, srcPckt.buf)

	// Copy all data
	newPckt.lastIndex = srcPckt.lastIndex
	newPckt.transportPacket = srcPckt.transportPacket
	newPckt.pat = srcPckt.pat

	newPckt.pmt.AudioADTS = make([]uint16, len(srcPckt.pmt.AudioADTS))
	copy(newPckt.pmt.AudioADTS, srcPckt.pmt.AudioADTS)
	newPckt.pmt.Videoh264 = make([]uint16, len(srcPckt.pmt.Videoh264))
	copy(newPckt.pmt.Videoh264, srcPckt.pmt.Videoh264)
	newPckt.pmt.Other = make([]uint16, len(srcPckt.pmt.Other))
	copy(newPckt.pmt.Other, srcPckt.pmt.Other)
	newPckt.pmt.valid = srcPckt.pmt.valid

	return newPckt
}

// Reset packet
func (p *TsPacket) Reset() {
	p.lastIndex = 0
	p.transportPacket.Reset()
}

// AddData Adds bytes to the packet
func (p *TsPacket) AddData(buf []byte) {

	p.lastIndex = p.lastIndex + copy(p.buf[p.lastIndex:], buf[:])
}

// GetBuffer Gets the buffer
func (p *TsPacket) GetBuffer() []byte {
	ret := make([]byte, len(p.buf))
	copy(ret, p.buf)
	return ret
}

// IsComplete Adds bytes to the packet
func (p *TsPacket) IsComplete() bool {
	if p.lastIndex == TsDefaultPacketSize && p.buf[0] == tsStartByte {
		return true
	}
	return false
}

// Parse Parse the packet
func (p *TsPacket) Parse(pmtPID int) bool {
	if !p.IsComplete() {
		return false
	}

	var transportPacket struct {
		SyncByte                      uint8
		ErrorIndicatorPayloadUnitPid  uint16
		ScrambledAdapFieldContCounter uint8
	}

	r := bytes.NewReader(p.buf)
	err := binary.Read(r, binary.BigEndian, &transportPacket)
	if err != nil {
		return false
	}
	p.transportPacket.Reset()

	p.transportPacket.SyncByte = transportPacket.SyncByte
	if transportPacket.ErrorIndicatorPayloadUnitPid&0x8000 > 0 {
		p.transportPacket.TransportErrorIndicator = true
	}
	if transportPacket.ErrorIndicatorPayloadUnitPid&0x4000 > 0 {
		p.transportPacket.PayloadUnitStartIndicator = true
	}
	if transportPacket.ErrorIndicatorPayloadUnitPid&0x2000 > 0 {
		p.transportPacket.TransportPriority = true
	}
	p.transportPacket.PID = transportPacket.ErrorIndicatorPayloadUnitPid & 0x1FFF

	p.transportPacket.TransportScramblingControl = (transportPacket.ScrambledAdapFieldContCounter & 0xC0) >> 6
	p.transportPacket.AdaptationFieldControl = (transportPacket.ScrambledAdapFieldContCounter & 0x30) >> 4
	p.transportPacket.ContinuityCounter = transportPacket.ScrambledAdapFieldContCounter & 0x0F

	if p.transportPacket.AdaptationFieldControl == 2 || p.transportPacket.AdaptationFieldControl == 3 {
		var adaptationFieldLength uint8
		err := binary.Read(r, binary.BigEndian, &adaptationFieldLength)
		if err != nil {
			return false
		}

		if adaptationFieldLength > 0 {
			var adaptationFieldFlags uint8
			err := binary.Read(r, binary.BigEndian, &adaptationFieldFlags)
			if err != nil {
				return false
			}
			if (adaptationFieldFlags & 0x80) > 0 {
				p.transportPacket.AdaptationField.DiscontinuityIndicator = true
			}
			if (adaptationFieldFlags & 0x40) > 0 {
				p.transportPacket.AdaptationField.RandomAccessIndicator = true
			}
			if (adaptationFieldFlags & 0x20) > 0 {
				p.transportPacket.AdaptationField.ElementaryStreamPriorityIndicator = true
			}
			if (adaptationFieldFlags & 0x10) > 0 {
				p.transportPacket.AdaptationField.PCRFlag = true
			}
			if (adaptationFieldFlags & 0x08) > 0 {
				p.transportPacket.AdaptationField.OPCRFlag = true
			}
			if (adaptationFieldFlags & 0x04) > 0 {
				p.transportPacket.AdaptationField.SplicingPointFlag = true
			}
			if (adaptationFieldFlags & 0x02) > 0 {
				p.transportPacket.AdaptationField.TransportPrivateDataFlag = true
			}
			if (adaptationFieldFlags & 0x01) > 0 {
				p.transportPacket.AdaptationField.AdaptationFieldExtensionFlag = true
			}

			if p.transportPacket.AdaptationField.PCRFlag == true {
				var pcrDataFirst32b uint32
				err := binary.Read(r, binary.BigEndian, &pcrDataFirst32b)
				if err != nil {
					return false
				}

				var pcrDataLast16b uint16
				err = binary.Read(r, binary.BigEndian, &pcrDataLast16b)
				if err != nil {
					return false
				}
				p.transportPacket.AdaptationField.PCRData.ProgramClockReferenceExtension = uint16(pcrDataLast16b & 0x1FF)
				p.transportPacket.AdaptationField.PCRData.reserved = uint8((pcrDataLast16b >> 9) & 0x3F)

				p.transportPacket.AdaptationField.PCRData.ProgramClockReferenceBase = uint64(pcrDataFirst32b)*2 + uint64((pcrDataLast16b>>15)&0x1)

				p.transportPacket.AdaptationField.PCRData.PCRs = calculatePCRS(p.transportPacket.AdaptationField.PCRData.ProgramClockReferenceBase, p.transportPacket.AdaptationField.PCRData.ProgramClockReferenceExtension)

				p.transportPacket.AdaptationField.PCRData.valid = true
			}
		}
	}

	if p.transportPacket.PID == PATPID || int(p.transportPacket.PID) == pmtPID {
		if p.transportPacket.PayloadUnitStartIndicator {
			var length uint8

			err = binary.Read(r, binary.BigEndian, &length)
			if err != nil {
				return false
			}

			var n uint8
			var dummyByte uint8
			for n < length {
				err = binary.Read(r, binary.BigEndian, &dummyByte)
				if err != nil {
					return false
				}

				n++
			}
		}
	}

	// PAT Packet (Getting the 1st PMT info, so assuming only one program and there is NO network ID)
	if p.transportPacket.PID == PATPID {
		var pmtPID16b struct {
			TableID                    uint8
			FlagsReservedSectionLength uint16
			TSId                       uint16
			Flags                      uint8
			SectionNumber              uint8
			LastSectionNumber          uint8

			//Initial PMT
			ProgramNumber  uint16
			ReservedPMTPID uint16
		}
		err = binary.Read(r, binary.BigEndian, &pmtPID16b)
		if err != nil {
			return false
		}
		p.transportPacket.Pat.PmtPID = pmtPID16b.ReservedPMTPID & 0x1FFF
		p.transportPacket.Pat.valid = true
	}

	// PMT Packet
	if pmtPID == int(p.transportPacket.PID) {
		var tableInfo struct {
			_                uint8
			SectionLength    uint16
			_                uint32
			_                uint16
			_                uint8
			ProgamInfoLength uint16
		}
		err = binary.Read(r, binary.BigEndian, &tableInfo)
		if err != nil {
			return false
		}

		sectionLength := tableInfo.SectionLength & 0x0FFF
		tableEnd := int(sectionLength - 13)

		programInfoLength := tableInfo.ProgamInfoLength & 0x0FFF

		paddingBytes := programInfoLength
		offset := 0

		for offset < tableEnd {
			for paddingBytes > 0 {
				var pad uint8
				err = binary.Read(r, binary.BigEndian, &pad)
				if err != nil {
					return false
				}
				paddingBytes = paddingBytes - 1
				offset = offset + 1
			}
			var program struct {
				StreamType uint8
				PID        uint16
				Next       uint16
			}
			err = binary.Read(r, binary.BigEndian, &program)
			if err != nil {
				return false
			}
			offset = offset + 5

			paddingBytes = program.Next & 0x0FFF
			pid := program.PID & 0x1FFF

			switch program.StreamType {
			case H264StreamType:
				p.transportPacket.Pmt.Videoh264 = append(p.transportPacket.Pmt.Videoh264, pid)
			case ADTSStreamType:
				p.transportPacket.Pmt.AudioADTS = append(p.transportPacket.Pmt.AudioADTS, pid)
			default:
				p.transportPacket.Pmt.Other = append(p.transportPacket.Pmt.Other, pid)
			}

			p.transportPacket.Pmt.valid = true
		}
	}

	p.transportPacket.valid = true

	return true
}

func calculatePCRS(pcrBase uint64, pcrExtension uint16) (PCRs float64) {
	PCRs = -1

	if pcrExtension > 0 {
		PCRs = float64(pcrBase*300.0+uint64(pcrExtension)) / (27.0 * 1000000.0)
	} else {
		PCRs = float64(pcrBase) / 90000.0
	}

	return
}

// GetPCRS Get PCT in seconds
func (p *TsPacket) GetPCRS() (PCRs float64) {
	PCRs = -1
	if !p.transportPacket.valid || !p.transportPacket.AdaptationField.PCRData.valid {
		return
	}

	PCRs = p.transportPacket.AdaptationField.PCRData.PCRs

	return
}

// GetPATdata Gets the PAT info if present (so PMT PID)
func (p *TsPacket) GetPATdata() (PMTPID int) {
	PMTPID = -1
	if !p.transportPacket.valid || !p.transportPacket.Pat.valid {
		return
	}

	PMTPID = int(p.transportPacket.Pat.PmtPID)

	return
}

// GetPMTdata Gets the PMT dta if present (video, audios, and other PIDs)
func (p *TsPacket) GetPMTdata() (valid bool, Videoh264 []uint16, AudioADTS []uint16, Other []uint16) {
	valid = false
	if !p.transportPacket.valid || !p.transportPacket.Pmt.valid {
		return
	}

	Videoh264 = p.transportPacket.Pmt.Videoh264
	AudioADTS = p.transportPacket.Pmt.AudioADTS
	Other = p.transportPacket.Pmt.Other
	valid = true

	return
}

// GetPID Adds bytes to the packet
func (p *TsPacket) GetPID() (pID int) {
	pID = -1
	if !p.transportPacket.valid {
		return
	}

	pID = int(p.transportPacket.PID)

	return
}

// String retuns packet data in string form
func (p *TsPacket) String() string {
	ret := ""
	if !p.transportPacket.valid {
		return ret
	}

	ret = fmt.Sprintf("%+v", p.transportPacket)
	return ret
}

// IsRandomAccess Return true if this is a random access point
func (p *TsPacket) IsRandomAccess(pID int) (isIDR bool) {
	isIDR = false
	if !p.transportPacket.valid {
		return
	}

	if p.transportPacket.PID == uint16(pID) {
		if p.transportPacket.AdaptationFieldControl == 2 || p.transportPacket.AdaptationFieldControl == 3 {
			if p.transportPacket.AdaptationField.RandomAccessIndicator == true {
				isIDR = true
			}
		}
	}

	return
}
