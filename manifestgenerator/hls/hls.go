package hls

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jordicenzano/go-ts-segmenter/uploaders/httpuploader"
	"github.com/jordicenzano/go-ts-segmenter/uploaders/s3uploader"
	"github.com/sirupsen/logrus"
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

// OutputTypes indicates the manifest type
type OutputTypes int

const (
	// HlsOutputModeNone No no write data
	HlsOutputModeNone OutputTypes = iota

	// HlsOutputModeFile Saves data to file
	HlsOutputModeFile

	// HlsOutputModeHTTP data to HTTP streaming server
	HlsOutputModeHTTP

	// HlsOutputModeS3 data to S3 (using AWS API)
	HlsOutputModeS3
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
	log                   *logrus.Logger
	manifestType          ManifestTypes
	version               int
	isIndependentSegments bool
	targetDurS            float64
	slidingWindowSize     int
	mseq                  int64
	dseq                  int64
	chunks                []Chunk
	chunklistFileName     string
	initChunkDataFileName string
	outputType            OutputTypes
	httpUploader          *httpuploader.HTTPUploader
	s3Uploader            *s3uploader.S3Uploader
	isClosed              bool
}

// New Creates a hls chunklist manifest
func New(
	log *logrus.Logger,
	ManifestType ManifestTypes,
	version int,
	isIndependentSegments bool,
	targetDurS float64,
	slidingWindowSize int,
	chunklistFileName string,
	initChunkDataFileName string,
	outputType OutputTypes,
	httpUploader *httpuploader.HTTPUploader,
	s3Uploader *s3uploader.S3Uploader,
) Hls {
	h := Hls{
		log,
		ManifestType,
		version,
		isIndependentSegments,
		targetDurS,
		slidingWindowSize,
		0,
		0,
		make([]Chunk, 0),
		chunklistFileName,
		initChunkDataFileName,
		outputType,
		httpUploader,
		s3Uploader,
		false,
	}

	return h
}

// SetInitChunk Adds a chunk init infomation
func (p *Hls) SetInitChunk(initChunkFileName string) {
	p.initChunkDataFileName = initChunkFileName
}

func (p *Hls) saveChunklist() error {
	ret := error(nil)

	hlsStrByte := []byte(p.String())

	if p.outputType == HlsOutputModeFile {
		ret = p.saveManifestToFile(hlsStrByte)
	} else if p.outputType == HlsOutputModeHTTP || p.outputType == HlsOutputModeS3 {
		ret = p.saveManifestExternal(hlsStrByte, p.outputType)
	}
	return ret
}

// CloseManifest Adds a chunk init infomation
func (p *Hls) CloseManifest(saveChunklist bool) error {
	ret := error(nil)

	p.isClosed = true

	if saveChunklist {
		ret = p.saveChunklist()
	}

	return ret
}

// SetHlsVersion Sets manifest version
func (p *Hls) SetHlsVersion(version int) {
	p.version = version
}

func (p *Hls) saveManifestToFile(manifestByte []byte) error {
	if p.chunklistFileName != "" {
		err := ioutil.WriteFile(p.chunklistFileName, manifestByte, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Hls) saveManifestExternal(manifestByte []byte, outputType OutputTypes) error {
	if p.chunklistFileName != "" {
		h := make(map[string]string)
		if strings.ToLower(path.Ext(p.chunklistFileName)) == ".m3u8" {
			h["Content-Type"] = "application/vnd.apple.mpegurl"
		}

		// TODO: Use interfaces
		if outputType == HlsOutputModeS3 {
			return p.s3Uploader.UploadData(manifestByte, p.chunklistFileName, h)
		}
		return p.httpUploader.UploadData(manifestByte, p.chunklistFileName, h)
	}
	return nil
}

// AddChunk Adds a new chunk
func (p *Hls) AddChunk(chunkData Chunk, saveChunklist bool) error {
	ret := error(nil)

	p.chunks = append(p.chunks, chunkData)

	if p.manifestType == LiveWindow && len(p.chunks) > p.slidingWindowSize {
		//Remove first
		if p.chunks[0].IsDisco {

		}
		p.chunks = p.chunks[1:]
		p.mseq++
	}

	if saveChunklist {
		ret = p.saveChunklist()
	}

	return ret
}

// addChunk Adds a new chunk
func (p *Hls) String() string {
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

	if p.isClosed {
		buffer.WriteString("#EXT-X-ENDLIST\n")
	}

	return buffer.String()
}
