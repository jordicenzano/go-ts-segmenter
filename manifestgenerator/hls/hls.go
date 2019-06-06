package hls

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"

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
	// OutputModeNone No no write data
	OutputModeNone OutputTypes = iota

	// OutputModeFile Saves chunks to file
	OutputModeFile

	// OutputModeHttp chunks to chunked streaming server
	OutputModeHttp
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
	outputType            OutputTypes
	httpClient            *http.Client
	httpScheme            string
	httpHost              string
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
	outputType OutputTypes,
	httpClient *http.Client,
	httpScheme string,
	httpHost string,
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
		outputType,
		httpClient,
		httpScheme,
		httpHost,
	}

	return h
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
		hlsStr := p.String()

		hlsStrByte := []byte(hlsStr)

		if p.outputType == OutputModeFile {
			if p.chunklistFileName != "" {
				err := ioutil.WriteFile(p.chunklistFileName, hlsStrByte, 0644)
				if err != nil {
					return err
				}
			}
		}

		if p.outputType == OutputModeHttp {
			if p.chunklistFileName != "" {
				req := &http.Request{
					Method: "POST",
					URL: &url.URL{
						Scheme: p.httpScheme,
						Host:   p.httpHost,
						Path:   "/" + p.chunklistFileName,
					},
					ProtoMajor:    1,
					ProtoMinor:    1,
					ContentLength: -1,
					Body:          ioutil.NopCloser(bytes.NewReader(hlsStrByte)),
				}

				_, err := p.httpClient.Do(req)

				if err != nil {
					p.log.Error("Error uploading ", p.chunklistFileName, ". Error: ", err)
				} else {
					p.log.Debug("Upload of ", p.chunklistFileName, " complete")
				}
			}
		}
	}

	return nil
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

	if p.version < 3 {
		buffer.WriteString("#EXT-X-TARGETDURATION:" + fmt.Sprintf("%.0f", p.targetDurS) + "\n")
	} else {
		buffer.WriteString("#EXT-X-TARGETDURATION:" + fmt.Sprintf("%.8f", p.targetDurS) + "\n")
	}

	if p.isIndependentSegments {
		buffer.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
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
