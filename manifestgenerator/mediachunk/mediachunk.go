package mediachunk

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

// OutputTypes indicates the manifest type
type OutputTypes int

const (
	// None No no write data
	None OutputTypes = iota

	// File Saves chunks to file
	File
)

// Options Chunking options
type Options struct {
	OutputTo           OutputTypes
	LHLS               bool
	EstimatedDurationS float64
	FileNumberLength   int
	GhostPrefix        string
	FileExtension      string
	BasePath           string
	ChunkBaseFilename  string
}

// Chunk Chunk class
type Chunk struct {
	fileWritter    *bufio.Writer
	fileDescriptor *os.File
	options        Options

	index         uint64
	filename      string
	filenameGhost string
}

// New Creates a chunk instance
func New(index uint64, options Options) Chunk {
	c := Chunk{nil, nil, options, index, "", ""}

	c.filename = c.createFilename(options.BasePath, options.ChunkBaseFilename, index, options.FileNumberLength, options.FileExtension, "")
	if options.GhostPrefix != "" {
		c.filenameGhost = c.createFilename(options.BasePath, options.ChunkBaseFilename, index, options.FileNumberLength, options.FileExtension, options.GhostPrefix)
	}

	return c
}

//InitializeChunk Initializes chunk
func (c *Chunk) InitializeChunk() error {
	if c.options.OutputTo == File {
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
			// Create ghost file
			exists, _ := fileExists(c.filename)
			if !exists {
				var err error
				c.fileDescriptor, err = os.Create(c.filename)
				if err != nil {
					return err
				}

				c.fileWritter = bufio.NewWriter(c.fileDescriptor)
			}
		}
	}

	return nil
}

//Close Closes chunk
func (c *Chunk) Close() {
	if c.options.OutputTo == File {
		if c.filenameGhost != "" {
			exists, _ := fileExists(c.filenameGhost)
			if exists {
				os.Remove(c.filenameGhost)
			}
		}

		if c.fileWritter != nil {
			c.fileDescriptor.Close()
		}
	}

	return
}

//AddData Add data to chunk and flush it
func (c *Chunk) AddData(buf []byte) error {
	if c.options.OutputTo == File {
		if c.fileWritter != nil {
			writtenBytes, err := c.fileWritter.Write(buf)
			if writtenBytes != len(buf) && err != nil {
				return err
			}
		}

		c.fileWritter.Flush()
	}

	return nil
}

/*
//AddTSPacket Add data to chunk and flush it
func (c *Chunk) AddTSPacket(tsPckt tspacket.TsPacket) error {
	if c.options.OutputTo == File {
		if c.fileWritter != nil {
			buf := tsPckt.GetBuffer()
			writtenBytes, err := c.fileWritter.Write(buf)
			if writtenBytes != len(buf) && err != nil {
				return err
			}
		}

		c.fileWritter.Flush()
	}

	return nil
}*/

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

func (c *Chunk) createFilename(basePath string, chunkBaseFilename string, index uint64, fileNumberLength int, fileExtension string, ghostPrefix string) string {
	ret := ""
	if ghostPrefix != "" {
		ret = path.Join(basePath, ghostPrefix+chunkBaseFilename+padNumberWithZero(index, fileNumberLength)+fileExtension)
	} else {
		ret = path.Join(basePath, chunkBaseFilename+padNumberWithZero(index, fileNumberLength)+fileExtension)
	}

	return ret
}
