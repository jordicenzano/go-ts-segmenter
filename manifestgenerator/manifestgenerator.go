package manifestgenerator

import (
	"fmt"
	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator/tspacket"
	"github.com/sirupsen/logrus"
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
	log                *logrus.Logger
	isCreatingChunks   bool
	baseOutPath        string
	chunkBaseFilename  string
	targetSegmentDurS  float64
	videoPID           int
	audioPID           int
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
func New(log *logrus.Logger, isCreatingChunks bool, baseOutPath string, chunkBaseFilename string, targetSegmentDurS float64, videoPID int, audioPID int, manifestType ManifestTypes, liveWindowSize int, lhlsAdvancedChunks int) ManifestGenerator {
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.DebugLevel)
	}
	e := ManifestGenerator{options{log, isCreatingChunks, baseOutPath, chunkBaseFilename, targetSegmentDurS, videoPID, audioPID, manifestType, liveWindowSize, lhlsAdvancedChunks}, false, 0, tspacket.New(tspacket.TsDefaultPacketSize), 0}
	return e
}

// Test test
func (mg ManifestGenerator) Test() {
	mg.options.log.Info(Version)
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

func (mg *ManifestGenerator) processPacket() bool {
	if !mg.tsPacket.Parse() {
		return false
	}

	pID := mg.tsPacket.GetPID()
	if pID == mg.options.videoPID {
		mg.options.log.Debug("VIDEO: ", mg.tsPacket.ToString())
		//TODO JOC
	} else if pID == mg.options.audioPID {
		mg.options.log.Debug("AUDIO: ", mg.tsPacket.ToString())
		//TODO JOC
	} else if pID >= 0 {
		mg.options.log.Debug("OTHER: ", mg.tsPacket.ToString())
	} else {
		fmt.Println("OUT OF SYNC!!!")
		return false
	}

	return true
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
		if mg.processPacket() == false {
			mg.isInSync = false
		} else {
			mg.bytesToNextSync = tspacket.TsDefaultPacketSize
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
