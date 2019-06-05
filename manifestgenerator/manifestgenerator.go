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

const (
	// ChunkLengthToleranceS Tolerance calculationg chunk length
	ChunkLengthToleranceS = 0.25
)

type options struct {
	log                *logrus.Logger
	isCreatingChunks   bool
	baseOutPath        string
	chunkBaseFilename  string
	targetSegmentDurS  float64
	autoPIDs           bool
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
	detectedPMTID   int

	// Current TS packet data
	tsPacket tspacket.TsPacket

	// Time counters
	chunkStartTimeS float64
	lastPCRS        float64

	// Packet counter
	processedPackets uint64
}

// New Creates a chunklistgenerator instance
func New(log *logrus.Logger, isCreatingChunks bool, baseOutPath string, chunkBaseFilename string, targetSegmentDurS float64, autoPIDs bool, videoPID int, audioPID int, manifestType ManifestTypes, liveWindowSize int, lhlsAdvancedChunks int) ManifestGenerator {
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.DebugLevel)
	}
	e := ManifestGenerator{options{log, isCreatingChunks, baseOutPath, chunkBaseFilename, targetSegmentDurS, autoPIDs, videoPID, audioPID, manifestType, liveWindowSize, lhlsAdvancedChunks}, false, 0, -1, tspacket.New(tspacket.TsDefaultPacketSize), -1.0, -1.0, 0}
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

func (mg *ManifestGenerator) processPacket(forceChunk bool) bool {
	if !mg.tsPacket.Parse(mg.detectedPMTID) {
		return false
	}

	if mg.options.autoPIDs {
		pmtID := mg.tsPacket.GetPATdata()
		if pmtID >= 0 {
			mg.detectedPMTID = pmtID
			mg.options.log.Debug("Detected PAT. PMT ID: ", pmtID)
		}

		valid, Videoh264, AudioADTS, Other := mg.tsPacket.GetPMTdata()
		if valid {
			if len(Videoh264) > 0 {
				mg.options.videoPID = int(Videoh264[0])
			}
			if len(AudioADTS) > 0 {
				mg.options.audioPID = int(AudioADTS[0])
			}

			mg.options.log.Debug("Detected PMT. VideoIDs: ", Videoh264, "AudiosIDs: ", AudioADTS, "Other: ", Other)
		}
	}

	pID := mg.tsPacket.GetPID()

	if pID == mg.options.videoPID {
		mg.options.log.Debug("VIDEO: ", mg.tsPacket.String())
		pcrS := mg.tsPacket.GetPCRS()
		if pcrS >= 0 {
			mg.lastPCRS = pcrS

			if mg.chunkStartTimeS < 0 && pcrS >= 0 {
				mg.chunkStartTimeS = pcrS
			}
			durS := pcrS - mg.chunkStartTimeS
			if (durS + ChunkLengthToleranceS) > mg.options.targetSegmentDurS {
				//TODO: Chunk
				_, nextInitialPCRS := mg.nextChunk(pcrS, durS, tspacket.MaxPCRSValue)

				mg.chunkStartTimeS = nextInitialPCRS
			}
		}
	} else if pID == mg.options.audioPID {
		mg.options.log.Debug("AUDIO: ", mg.tsPacket.String())
		//TODO JOC
	} else if pID >= 0 {
		mg.options.log.Debug("OTHER: ", mg.tsPacket.String())
	} else {
		fmt.Println("OUT OF SYNC!!!")
		return false
	}

	return true
}

// Creates chunk and returns the initial time for the next chunk
func (mg *ManifestGenerator) nextChunk(currentPCRS float64, lastInitialPCRS float64, maxPCRs float64) (chunkDurationS float64, nextInitialPCRS float64) {
	chunkDurationS = -1.0
	nextInitialPCRS = currentPCRS

	if currentPCRS >= lastInitialPCRS {
		chunkDurationS = currentPCRS - lastInitialPCRS
	} else {
		// Detected possible PCR roll over
		mg.options.log.Info("Possible PCR rollover! lastInitialPCRS:", lastInitialPCRS, ", currentPCRS: ", currentPCRS, ", maxPCRs: ", maxPCRs)
		chunkDurationS = maxPCRs - currentPCRS + lastInitialPCRS
	}

	mg.options.log.Info("CHUNK! At PCRs: ", currentPCRS, ". ChunkDurS: ", chunkDurationS)

	return
}

// Close Closes manigest processing saving last data and last chunk
func (mg *ManifestGenerator) Close() {
	//Generate last chunk
	mg.nextChunk(mg.lastPCRS, mg.chunkStartTimeS, tspacket.MaxPCRSValue)
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
		if mg.processPacket(false) == false {
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

func (mg ManifestGenerator) getNumProcessedPackets() uint64 {
	return mg.processedPackets
}
