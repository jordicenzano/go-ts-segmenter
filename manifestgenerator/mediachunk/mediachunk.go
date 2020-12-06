package mediachunk

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jordicenzano/go-ts-segmenter/uploaders/httpuploader"
	"github.com/jordicenzano/go-ts-segmenter/uploaders/s3uploader"
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

	// ChunkOutputModeS3 chunks to S3
	ChunkOutputModeS3
)

// Options Chunking options
type Options struct {
	Log                *logrus.Logger
	OutputType         OutputTypes
	LHLS               bool
	EstimatedDurationS float64
	FileNumberLength   int
	GhostPrefix        string
	FileExtension      string
	BasePath           string
	ChunkBaseFilename  string
	HTTPUploader       *httpuploader.HTTPUploader
	S3Uploader         *s3uploader.S3Uploader
}

// Chunk Chunk class
type Chunk struct {
	// Used by HTTP chunked based
	httpWriteChan chan<- []byte

	// Used by file chunk write
	fileWriter     *bufio.Writer
	fileDescriptor *os.File

	options Options

	index         uint64
	filename      string
	filenameGhost string

	// Used by NON chunk transfer HTTP
	tmpFilename string

	// Bytes received
	totalBytes int

	// Epoch time when we received first byte for this chunk
	createdAt int64
}

// New Creates a chunk instance
func New(index uint64, options Options) Chunk {
	c := Chunk{nil, nil, nil, options, index, "", "", "", 0, time.Now().UnixNano()}

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

func (c *Chunk) initializeChunkHTTPChunkedTransfer() error {
	c.httpWriteChan = c.options.HTTPUploader.UploadChunkedTransfer(c.filename, c.getChunkHeaders())

	return nil
}

//InitializeChunk Initializes chunk
func (c *Chunk) InitializeChunk() error {
	ret := error(nil)

	if c.options.OutputType == ChunkOutputModeFile {
		ret = c.initializeChunkFile()
	} else if c.options.OutputType == ChunkOutputModeHTTPChunkedTransfer {
		ret = c.initializeChunkHTTPChunkedTransfer()
	} else if c.options.OutputType == ChunkOutputModeHTTPRegular || c.options.OutputType == ChunkOutputModeS3 {
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

func (c *Chunk) closeChunkTmpFileExternal(outputType OutputTypes) {
	if c.fileWriter != nil {
		c.fileDescriptor.Sync()
		c.fileDescriptor.Close()
	}

	if c.tmpFilename != "" {
		h := c.getChunkHeaders()
		if outputType == ChunkOutputModeS3 {
			c.options.S3Uploader.UploadLocalFile(c.tmpFilename, c.filename, h)
		} else {
			c.options.HTTPUploader.UploadLocalFile(c.tmpFilename, c.filename, h)
		}
	}

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
	} else if c.options.OutputType == ChunkOutputModeHTTPRegular || c.options.OutputType == ChunkOutputModeS3 {
		c.closeChunkTmpFileExternal(c.options.OutputType)
	}
	return
}

func (c *Chunk) getChunkHeaders() map[string]string {
	h := make(map[string]string)
	if strings.ToLower(path.Ext(c.filename)) == ".ts" {
		h["Content-Type"] = "video/MP2T"
		h["Joc-Hls-Chunk-Seq-Number"] = strconv.FormatUint(c.index, 10)
		h["Joc-Hls-Targetduration-Ms"] = strconv.FormatFloat(c.options.EstimatedDurationS*1000, 'f', 8, 64)
		h["Joc-Hls-CreatedAt-Ns"] = strconv.FormatInt(c.createdAt, 10)
	}
	return h
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

	if c.options.OutputType == ChunkOutputModeFile || c.options.OutputType == ChunkOutputModeHTTPRegular || c.options.OutputType == ChunkOutputModeS3 {
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
