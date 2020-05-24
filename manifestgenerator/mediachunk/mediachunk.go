package mediachunk

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// OutputTypes indicates the manifest type
type OutputTypes int

const (
	// ChunkOutputModeNone No no write data
	ChunkOutputModeNone OutputTypes = iota

	// ChunkOutputModeFile Saves chunks to file
	ChunkOutputModeFile

	// ChunkOutputModeHTTPChunkedTransfer chunks to chunked streaming server
	ChunkOutputModeHTTPChunkedTransfer

	// ChunkOutputModeHTTPRegular chunks to chunked streaming server
	ChunkOutputModeHTTPRegular
)

// Options Chunking options
type Options struct {
	Log                     *logrus.Logger
	OutputType              OutputTypes
	LHLS                    bool
	EstimatedDurationS      float64
	FileNumberLength        int
	GhostPrefix             string
	FileExtension           string
	BasePath                string
	ChunkBaseFilename       string
	HTTPClient              *http.Client
	HTTPScheme              string
	HTTPHost                string
	MaxHTTPRetries          int
	InitialHTTPRetryDelayMs int
}

// Chunk Chunk class
type Chunk struct {
	fileWriter     *bufio.Writer
	fileDescriptor *os.File

	httpWriteChan chan<- []byte
	httpReq       *http.Request

	options Options

	index         uint64
	filename      string
	filenameGhost string

	// Used by NON chunk transfer HTTP
	tmpFilename string

	totalBytes int
}

// New Creates a chunk instance
func New(index uint64, options Options) Chunk {
	c := Chunk{nil, nil, nil, nil, options, index, "", "", "", 0}

	c.filename = c.createFilename(options.BasePath, options.ChunkBaseFilename, index, options.FileNumberLength, options.FileExtension, "")
	if options.GhostPrefix != "" {
		c.filenameGhost = c.createFilename(options.BasePath, options.ChunkBaseFilename, index, options.FileNumberLength, options.FileExtension, options.GhostPrefix)
	}

	return c
}

func (c *Chunk) initializeChunkTempFile() error {
	rand.Seed(time.Now().UnixNano())
	c.tmpFilename = filepath.Join(os.TempDir(), strconv.Itoa(rand.Intn(1<<32-1))+".tmp")
	// Create media file
	exists, _ := fileExists(c.tmpFilename)
	if !exists {
		var err error
		c.fileDescriptor, err = os.Create(c.tmpFilename)
		if err != nil {
			return err
		}

		c.fileWriter = bufio.NewWriter(c.fileDescriptor)
	}

	return nil
}

func (c *Chunk) initializeChunkFile() error {
	if c.filenameGhost != "" {
		// Create ghost file
		exists, _ := fileExists(c.filenameGhost)
		if !exists {
			err := ioutil.WriteFile(c.filenameGhost, nil, 0644)
			if err != nil {
				return err
			}
		}
	}

	if c.filename != "" {
		// Create media file
		exists, _ := fileExists(c.filename)
		if !exists {
			var err error
			c.fileDescriptor, err = os.Create(c.filename)
			if err != nil {
				return err
			}

			c.fileWriter = bufio.NewWriter(c.fileDescriptor)
		}
	}

	return nil
}

func (c *Chunk) uploadChunkTmpFileToHTTPRetry() error {
	var ret error = nil
	maxRetries := c.options.MaxHTTPRetries
	retryPauseInitialMs := c.options.InitialHTTPRetryDelayMs
	retryIntent := 0

	for {
		if retryIntent >= maxRetries {
			c.options.Log.Error("ERROR chunk lost becasue server busy ", c.tmpFilename, "(", c.filename, ")")
			break
		} else {
			retrieableErr := c.uploadChunkTmpFileToHTTP()
			if retrieableErr != nil {
				time.Sleep(time.Duration(retryPauseInitialMs*retryIntent) * time.Millisecond)
			} else {
				break
			}
		}
		retryIntent++
	}

	return ret
}

func (c *Chunk) uploadChunkTmpFileToHTTP() error {
	var ret error = nil

	if c.tmpFilename != "" {
		f, errOpen := os.Open(c.tmpFilename)
		defer f.Close()
		if errOpen != nil {
			c.options.Log.Error("ERROR reading  ", c.tmpFilename, "(", c.filename, ")")
		} else {
			req := &http.Request{
				Method: "POST",
				URL: &url.URL{
					Scheme: c.options.HTTPScheme,
					Host:   c.options.HTTPHost,
					Path:   "/" + c.filename,
				},
				ProtoMajor:    1,
				ProtoMinor:    1,
				ContentLength: -1,
				Body:          ioutil.NopCloser(f),
				Header:        http.Header{},
			}

			if strings.ToLower(path.Ext(c.filename)) == ".ts" {
				req.Header.Set("Content-Type", "video/MP2T")
				req.Header.Set("Joc-Hls-Chunk-Seq-Number", strconv.FormatUint(c.index, 10))
				req.Header.Set("Joc-Hls-Targetduration-Ms", strconv.FormatFloat(c.options.EstimatedDurationS*1000, 'f', 8, 64))
			}

			resp, errReq := c.options.HTTPClient.Do(req)
			if errReq != nil {
				c.options.Log.Error("Error uploading ", c.tmpFilename, "(", c.filename, ")", "Error: ", errReq)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode < 400 {
					// Done
					c.options.Log.Info("Upload of ", c.tmpFilename, "(", c.filename, ") complete")
				} else if resp.StatusCode == http.StatusServiceUnavailable {
					// Need to retry
					c.options.Log.Debug("Warning server busy for ", c.tmpFilename, "(", c.filename, "), RETRYING!")
					ret = errors.New("Retryable upload error")
				} else {
					// Not retriable error
					c.options.Log.Error("Error server uploading ", c.tmpFilename, "(", c.filename, ")", "HTTP Error: ", resp.StatusCode)
				}
			}
		}
	}

	return ret
}

func (c *Chunk) initializeChunkHTTPChunkedTransfer() error {
	r, w := io.Pipe()
	writeChan := make(chan []byte)
	c.httpWriteChan = writeChan

	// open request
	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: c.options.HTTPScheme,
			Host:   c.options.HTTPHost,
			Path:   "/" + c.filename,
		},
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: -1,
		Body:          r,
		Header:        http.Header{},
	}

	if strings.ToLower(path.Ext(c.filename)) == ".ts" {
		req.Header.Set("Content-Type", "video/MP2T")
		req.Header.Set("Joc-Hls-Chunk-Seq-Number", strconv.FormatUint(c.index, 10))
		req.Header.Set("Joc-Hls-Targetduration-Ms", strconv.FormatFloat(c.options.EstimatedDurationS*1000, 'f', 8, 64))
	}
	c.httpReq = req

	go func() {
		defer w.Close()

		for buf := range writeChan {
			n, err := w.Write(buf)
			c.options.Log.Debug("Wrote ", n, " bytes to chunk ", c.filename)
			if n != len(buf) && err != nil {
				panic(err)
			}
		}
	}()

	go func() {
		c.options.Log.Debug("Opening connection to upload ", c.filename)
		c.options.Log.Debug("Req: ", req)
		_, err := c.options.HTTPClient.Do(req)

		if err != nil {
			c.options.Log.Error("Error uploading ", c.filename, ". Error: ", err)
		} else {
			c.options.Log.Debug("Upload of ", c.filename, " complete")
		}
	}()

	return nil
}

//InitializeChunk Initializes chunk
func (c *Chunk) InitializeChunk() error {
	ret := error(nil)

	if c.options.OutputType == ChunkOutputModeFile {
		ret = c.initializeChunkFile()
	} else if c.options.OutputType == ChunkOutputModeHTTPChunkedTransfer {
		ret = c.initializeChunkHTTPChunkedTransfer()
	} else if c.options.OutputType == ChunkOutputModeHTTPRegular {
		ret = c.initializeChunkTempFile()
	}

	return ret
}

func (c *Chunk) closeChunkFile() {
	if c.filenameGhost != "" {
		exists, _ := fileExists(c.filenameGhost)
		if exists {
			os.Remove(c.filenameGhost)
		}
	}

	if c.fileWriter != nil {
		c.fileDescriptor.Close()
	}
}

func (c *Chunk) closeChunkTmpFileToHTTP() {
	if c.fileWriter != nil {
		c.fileDescriptor.Sync()
		c.fileDescriptor.Close()
	}

	// Upload to HTTP server
	c.uploadChunkTmpFileToHTTPRetry()

	// Delete temp file
	exists, _ := fileExists(c.tmpFilename)
	if exists {
		os.Remove(c.tmpFilename)
	}
}

func (c *Chunk) closeChunkHTTPChunkedTransfer() {
	if c.httpWriteChan != nil {
		close(c.httpWriteChan)
	}
}

//Close Closes chunk
func (c *Chunk) Close() {
	c.options.Log.Debug("Closing chunk ", c.filename)
	if c.options.OutputType == ChunkOutputModeFile {
		c.closeChunkFile()
	} else if c.options.OutputType == ChunkOutputModeHTTPChunkedTransfer {
		c.closeChunkHTTPChunkedTransfer()
	} else if c.options.OutputType == ChunkOutputModeHTTPRegular {
		c.closeChunkTmpFileToHTTP()
	}

	return
}

func (c *Chunk) addDataChunkFile(buf []byte) error {
	if c.fileWriter != nil {
		totalWrittenBytes := 0
		err := error(nil)

		for totalWrittenBytes < len(buf) && err == nil {
			writtenBytes, err := c.fileWriter.Write(buf[totalWrittenBytes:])

			totalWrittenBytes = totalWrittenBytes + writtenBytes

			if err != nil {
				return err
			}
		}
		c.fileWriter.Flush()
	}

	return nil
}

func (c *Chunk) addDataChunkHTTP(buf []byte) error {
	if c.httpWriteChan != nil {
		bufCopy := make([]byte, len(buf))
		copy(bufCopy, buf)

		c.httpWriteChan <- bufCopy
	}
	return nil
}

//AddData Add data to chunk and flush it
func (c *Chunk) AddData(buf []byte) error {
	ret := error(nil)

	c.options.Log.Debug("Adding data to chunk ", c.filename)

	if c.options.OutputType == ChunkOutputModeFile || c.options.OutputType == ChunkOutputModeHTTPRegular {
		ret = c.addDataChunkFile(buf)
	} else if c.options.OutputType == ChunkOutputModeHTTPChunkedTransfer {
		ret = c.addDataChunkHTTP(buf)
	}

	c.totalBytes = c.totalBytes + len(buf)

	return ret
}

//IsEmpty Indicates if there are any bytes already saved in this chunk
func (c *Chunk) IsEmpty() bool {
	ret := true
	if c.totalBytes > 0 {
		ret = false
	}

	return ret
}

//GetFilename Returns the filename
func (c *Chunk) GetFilename() string {
	return c.filename
}

//GetIndex Returns the index
func (c *Chunk) GetIndex() uint64 {
	return c.index
}

func fileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func padNumberWithZero(value uint64, numZeros int) string {
	format := "%0" + strconv.Itoa(numZeros) + "d"
	return fmt.Sprintf(format, value)
}

func (c *Chunk) createFilename(
	basePath string,
	chunkBaseFilename string,
	index uint64,
	fileNumberLength int,
	fileExtension string,
	ghostPrefix string,
) string {
	ret := ""
	if ghostPrefix != "" {
		ret = path.Join(basePath, ghostPrefix+chunkBaseFilename+padNumberWithZero(index, fileNumberLength)+fileExtension)
	} else {
		ret = path.Join(basePath, chunkBaseFilename+padNumberWithZero(index, fileNumberLength)+fileExtension)
	}

	return ret
}
