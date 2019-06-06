package manifestgenerator

import (
	"fmt"

	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator/mediachunk"
	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator/tspacket"
	"github.com/sirupsen/logrus"
)

// Version Indicates the package version
var Version = "1.1.0"

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
	//GhostPrefixDefault ghost chunk prefix
	GhostPrefixDefault = ".growing_"

	//ChunkFileNumberLength chunk filenumber length
	ChunkFileNumberLength = 5

	//ChunkFileExtensionDefault default chunk extension
	ChunkFileExtensionDefault = ".ts"

	//ChunkInitFileName Init chunk filename
	ChunkInitFileName = "init"
)

const (
	// ChunkLengthToleranceS Tolerance calculationg chunk length
	ChunkLengthToleranceS = 0.25
)

// packetTableTypes
type packetTableTypes int

const (
	// PatTable PAT
	PatTable = iota

	// PmtTable PAT
	PmtTable
)

// initState
type initStates int

const (
	// InitNotIni no PAT / PMT saved
	InitNotIni = iota

	// InitsavedPAT PAT saved, needs PMT too
	InitsavedPAT

	// InitsavedPMT PMT and PAT saved
	InitsavedPMT
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

	//currentChunk info
	currentChunk      *mediachunk.Chunk
	currentChunkIndex uint64

	//currentChunk info
	initChunk *mediachunk.Chunk
	initState initStates
}

// New Creates a chunklistgenerator instance
func New(log *logrus.Logger, isCreatingChunks bool, baseOutPath string, chunkBaseFilename string, targetSegmentDurS float64, autoPIDs bool, videoPID int, audioPID int, manifestType ManifestTypes, liveWindowSize int, lhlsAdvancedChunks int) ManifestGenerator {
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.DebugLevel)
	}

	mg := ManifestGenerator{options{log, isCreatingChunks, baseOutPath, chunkBaseFilename, targetSegmentDurS, autoPIDs, videoPID, audioPID, manifestType, liveWindowSize, lhlsAdvancedChunks}, false, 0, -1, tspacket.New(tspacket.TsDefaultPacketSize), -1.0, -1.0, 0, nil, 0, nil, InitNotIni}

	return mg
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

func (mg *ManifestGenerator) isSavingMediaPacket() bool {
	ret := false
	if !mg.options.autoPIDs {
		ret = true
	} else {
		if mg.initState == InitsavedPMT {
			ret = true
		}
	}

	return ret
}

func (mg *ManifestGenerator) processPacket(forceChunk bool) bool {
	if !mg.tsPacket.Parse(mg.detectedPMTID) {
		return false
	}

	// Detect video & audio PIDs
	if mg.options.autoPIDs {
		pmtID := mg.tsPacket.GetPATdata()
		if pmtID >= 0 {
			mg.detectedPMTID = pmtID

			// Save PAT
			mg.addPacketToInitChunk(PatTable)

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

			// Save PMT
			mg.addPacketToInitChunk(PmtTable)

			mg.options.log.Debug("Detected PMT. VideoIDs: ", Videoh264, "AudiosIDs: ", AudioADTS, "Other: ", Other)
		}
	}

	pID := mg.tsPacket.GetPID()
	if pID == mg.options.videoPID {
		if mg.isSavingMediaPacket() {
			mg.addPacketToChunk()

			// Detect if we need to chunk it
			mg.options.log.Debug("VIDEO: ", mg.tsPacket.String())
			pcrS := mg.tsPacket.GetPCRS()
			if pcrS >= 0 {
				mg.lastPCRS = pcrS

				if mg.chunkStartTimeS < 0 && pcrS >= 0 {
					mg.chunkStartTimeS = pcrS
				}
				durS := pcrS - mg.chunkStartTimeS
				if (durS + ChunkLengthToleranceS) > mg.options.targetSegmentDurS {
					//TODO: Chunk First param is the real chunk durartion
					//TODO: JOC duration of chunk to HLS

					_, nextInitialPCRS := mg.nextChunk(pcrS, mg.chunkStartTimeS, tspacket.MaxPCRSValue, false)

					mg.chunkStartTimeS = nextInitialPCRS
				}
			}
		} else {
			mg.options.log.Debug("SKIPPED VIDEO PACKET, not init: ", mg.tsPacket.String())
		}
	} else if pID == mg.options.audioPID {
		if mg.isSavingMediaPacket() {
			mg.addPacketToChunk()
			mg.options.log.Debug("AUDIO: ", mg.tsPacket.String())
		} else {
			mg.options.log.Debug("SKIPPED AUDIO PACKET, not init: ", mg.tsPacket.String())
		}
	} else if pID >= 0 {
		mg.options.log.Debug("OTHER: ", mg.tsPacket.String())
	} else {
		fmt.Println("OUT OF SYNC!!!")
		return false
	}

	return true
}

func (mg *ManifestGenerator) addPacketToChunk() {

	if mg.currentChunk == nil {
		mg.createChunk(false)
	}

	err := mg.currentChunk.AddData(mg.tsPacket.GetBuffer())
	if err != nil {
		panic(err)
	}
}

func (mg *ManifestGenerator) addPacketToInitChunk(tableType packetTableTypes) bool {
	ret := false
	saveData := false

	if tableType == PatTable {
		if mg.initState == InitNotIni { // We only save the 1st PAT PMT appeareance, so no dynamic updates are allowed
			if mg.initChunk == nil {
				// Create init chunk
				mg.createChunk(true)
			}
			saveData = true
		}
	} else if tableType == PmtTable {
		if mg.initState == InitsavedPAT {
			saveData = true
		}
	}

	if saveData {
		err := mg.initChunk.AddData(mg.tsPacket.GetBuffer())
		if err != nil {
			panic(err)
		}

		if tableType == PatTable {
			mg.initState = InitsavedPAT
		} else if tableType == PmtTable {
			mg.initState = InitsavedPMT

			mg.closeChunk(true)
		}

		ret = true
	}

	return ret
}

func (mg *ManifestGenerator) closeChunk(isInit bool) {
	// Close current

	if isInit == false {
		if mg.currentChunk != nil {
			mg.currentChunk.Close()
			mg.currentChunk = nil
		}

		mg.currentChunkIndex++
	} else {
		if mg.initChunk != nil {
			mg.initChunk.Close()
			mg.initChunk = nil
		}
	}

	return
}

func (mg *ManifestGenerator) createChunk(isInit bool) {
	// Close current
	chunkOptions := mediachunk.Options{
		OutputTo:           mediachunk.File,
		LHLS:               false,
		EstimatedDurationS: mg.options.targetSegmentDurS,
		FileNumberLength:   ChunkFileNumberLength,
		GhostPrefix:        GhostPrefixDefault,
		FileExtension:      ChunkFileExtensionDefault,
		BasePath:           mg.options.baseOutPath,
		ChunkBaseFilename:  mg.options.chunkBaseFilename}

	if isInit {
		chunkOptions.ChunkBaseFilename = ChunkInitFileName
		chunkOptions.EstimatedDurationS = -1
		chunkOptions.LHLS = false

		newChunk := mediachunk.New(0, chunkOptions)
		mg.initChunk = &newChunk

		err := mg.initChunk.InitializeChunk()
		if err != nil {
			panic(err)
		}
	} else {
		newChunk := mediachunk.New(mg.currentChunkIndex, chunkOptions)
		mg.currentChunk = &newChunk

		err := mg.currentChunk.InitializeChunk()
		if err != nil {
			panic(err)
		}
	}
	return
}

// Creates chunk and returns the initial time for the next chunk
func (mg *ManifestGenerator) nextChunk(currentPCRS float64, lastInitialPCRS float64, maxPCRs float64, final bool) (chunkDurationS float64, nextInitialPCRS float64) {
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

	mg.closeChunk(false)
	if !final {
		mg.createChunk(false)
	}

	return
}

// Close Closes manigest processing saving last data and last chunk
func (mg *ManifestGenerator) Close() {
	//Generate last chunk
	mg.nextChunk(mg.lastPCRS, mg.chunkStartTimeS, tspacket.MaxPCRSValue, true)
}

// AddData current chunk
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
