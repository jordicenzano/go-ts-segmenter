package tspacket

/*
import (
	"bytes"
	"encoding/binary"
	"fmt"
)
*/

const (
	// TsDefaultPacketSize Default TS packet size
	TsDefaultPacketSize = 188

	// TsStartByte Start byte for TS pakcets
	TsStartByte = 0x47
)

// TsPacket Transport stream packet
type TsPacket struct {
	buf       []byte
	lastIndex int
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

// Process Indicates if packet is complete
func (p *TsPacket) Process() bool {
	if p.lastIndex == TsDefaultPacketSize {
		return p.parse()
	}

	return false
}

func (p *TsPacket) parse() bool {
	if p.buf[0] == TsStartByte {
		return true
	}

	return false
}
