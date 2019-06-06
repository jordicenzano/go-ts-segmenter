package hls

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
)

// Version Indicates the package version
var Version = "1.0.0"

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

// Chunk Chunk information
type Chunk struct {
	IsGrowing bool
	FileName  string
	DurationS float64
	IsDisco   bool
}

// Hls Hls chunklist
type Hls struct {
	manifestType          ManifestTypes
	version               int
	isIndependentSegments bool
	targetDurS            float64
	slidingWindowSize     int
	mseq                  int64
	dseq                  int64
	chunks                []Chunk

	chunklistFileName string

	initChunkDataFileName string
}

// New Creates a hls chunklist manifest
func New(ManifestType ManifestTypes, version int, isIndependentSegments bool, targetDurS float64, slidingWindowSize int, chunklistFileName string) Hls {
	h := Hls{ManifestType, version, isIndependentSegments, targetDurS, slidingWindowSize, 0, 0, make([]Chunk, 0), chunklistFileName, ""}

	return h
}

// AddInitChunk Adds a chunk init infomation
func (p *Hls) AddInitChunk(initChunkFileName string) {
	p.initChunkDataFileName = initChunkFileName
}

// SetHlsVersion Sets manifest version
func (p *Hls) SetHlsVersion(version int) {
	p.version = version
}

// AddChunk Adds a new chunk
func (p *Hls) AddChunk(chunkData Chunk, saveChunklist bool) error {

	p.chunks = append(p.chunks, chunkData)

	if p.manifestType == LiveWindow && len(p.chunks) > p.slidingWindowSize {
		//Remove first
		if p.chunks[0].IsDisco {

		}
		p.chunks = p.chunks[1:]
		p.mseq++
	}

	if saveChunklist {
		// Save chunklist file
		hlsStr := p.createHlsChunklist()

		hlsStrByte := []byte(hlsStr)

		if p.chunklistFileName != "" {
			err := ioutil.WriteFile(p.chunklistFileName, hlsStrByte, 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// addChunk Adds a new chunk
func (p *Hls) createHlsChunklist() string {
	var buffer bytes.Buffer

	buffer.WriteString("#EXTM3U\n")
	buffer.WriteString("#EXT-X-VERSION:" + strconv.Itoa(p.version) + "\n")
	buffer.WriteString("#EXT-X-MEDIA-SEQUENCE:" + strconv.FormatInt(p.mseq, 10) + "\n")
	buffer.WriteString("#EXT-X-DISCONTINUITY-SEQUENCE:" + strconv.FormatInt(p.dseq, 10) + "\n")

	if p.manifestType == Vod {
		buffer.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")

	} else if p.manifestType == LiveEvent {
		buffer.WriteString("#EXT-X-PLAYLIST-TYPE:EVENT\n")
	}

	buffer.WriteString("#EXT-X-TARGETDURATION:" + fmt.Sprintf("%.0f", p.targetDurS) + "\n")

	if p.isIndependentSegments {
		buffer.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
	}

	if p.initChunkDataFileName != "" {
		chunkPath, _ := filepath.Rel(path.Dir(p.chunklistFileName), p.initChunkDataFileName)
		buffer.WriteString("#EXT-X-MAP:URI=\"" + chunkPath + "\"\n")
	}

	for _, chunk := range p.chunks {
		if chunk.IsDisco {
			buffer.WriteString("#EXT-X-DISCONTINUITY\n")
		}
		buffer.WriteString("#EXTINF:" + fmt.Sprintf("%.8f", chunk.DurationS) + ",\n")

		chunkPath, _ := filepath.Rel(path.Dir(p.chunklistFileName), chunk.FileName)
		buffer.WriteString(chunkPath + "\n")
	}

	return buffer.String()
}
