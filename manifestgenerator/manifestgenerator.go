package manifestgenerator

import (
	"fmt"
	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator/tspacket"
)

// Version Indicates the package version
var Version = "1.0.2"

// ManifestTypes indicates the manifest type
type ManifestTypes int

const (
	// Vod Indicates VOD manifest
	Vod ManifestTypes = iota

	//LiveEvent Indicates a live manifest type event (always growing)
	LiveEvent

	//LiveWindow Indicates a live manifest type sliding window (fixed size)
	LiveWindow
)

type options struct {
	isCreatingChunks   bool
	baseOutPath        string
	chunkBaseFilename  string
	targetSegmentDurS  float64
	manifestType       ManifestTypes
	liveWindowSize     int
	lhlsAdvancedChunks int
}

// ManifestGenerator Creates the manifest and chunks the media
type ManifestGenerator struct {
	options options

	// Internal parsing data
	isInSync        bool
	bytesToNextSync int
	tsPacket        tspacket.TsPacket

	//TODO: del
	processedPackets int
}

// New Creates a chunklistgenerator instance
func New(isCreatingChunks bool, baseOutPath string, chunkBaseFilename string, targetSegmentDurS float64, manifestType ManifestTypes, liveWindowSize int, lhlsAdvancedChunks int) ManifestGenerator {
	e := ManifestGenerator{options{isCreatingChunks, baseOutPath, chunkBaseFilename, targetSegmentDurS, manifestType, liveWindowSize, lhlsAdvancedChunks}, false, 0, tspacket.New(tspacket.TsDefaultPacketSize), 0}
	return e
}

// Test test
func (mg ManifestGenerator) Test() {
	fmt.Printf("Test %v", mg)
}

func (mg *ManifestGenerator) resync(buf []byte) []byte {
	mg.isInSync = false

	start := 0
	for {
		if start < len(buf) {
			if buf[start] == 0x47 {
				mg.isInSync = true
				break
			} else {
				start++
			}
		} else {
			break
		}
	}

	return buf[start:]
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// AddData process recived data
func (mg *ManifestGenerator) AddData(buf []byte) {
	if !mg.isInSync {
		buf = mg.resync(buf)

		if len(buf) > 0 {
			mg.bytesToNextSync = tspacket.TsDefaultPacketSize
		}
	}

	if len(buf) > 0 {
		addedSize := min(len(buf), mg.bytesToNextSync)
		mg.tsPacket.AddData(buf[:addedSize])

		mg.bytesToNextSync = mg.bytesToNextSync - addedSize

		buf = buf[addedSize:]
	}

	if mg.bytesToNextSync <= 0 {
		// Process packet
		if mg.tsPacket.GetPID() < 0 {
			mg.isInSync = false
		} else {
			mg.processedPackets++
			mg.tsPacket.Reset()
		}
	}

	if len(buf) > 0 {
		// Still data to process
		mg.AddData(buf[:])
	}

	return
}

func (mg ManifestGenerator) getProcessedPackets() int {
	return mg.processedPackets
}
