package main

import (
	"flag"

	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator"
	"github.com/sirupsen/logrus"

	"bufio"
	"io"
	"os"
)

const (
	readBufferSize = 128
)

var (
	verbose            = flag.Bool("v", false, "enable to get verbose logging")
	isCreatingChunks   = flag.Bool("c", true, "Create chunks")
	baseOutPath        = flag.String("p", ".", "Output path")
	chunkBaseFilename  = flag.String("f", "chunk_", "Chunks base filename")
	targetSegmentDurS  = flag.Float64("t", 4.0, "Chunk duration in seconds")
	liveWindowSize     = flag.Int("w", 3, "Live window size in chunks")
	lhlsAdvancedChunks = flag.Int("l", 0, "LHLS advanced chunks")
	manifestTypeInt    = flag.Int("m", 2, "Manifest to generate (0- Vod, 1- Live event, 2- Live sliding window")
	videoPID           = flag.Int("vpid", 256, "Video PID to parse")
	audioPID           = flag.Int("apid", 257, "Audio PID to parse")
)

func main() {
	flag.Parse()

	var log = logrus.New()
	if *verbose {
		log.SetLevel(logrus.DebugLevel)
	}
	// TODO better path
	logPath := "./logs/server.log"

	Formatter := new(logrus.JSONFormatter)
	Formatter.TimestampFormat = "01-01-2001 13:00:00"
	log.SetFormatter(Formatter)
	log.SetFormatter(&logrus.JSONFormatter{})

	log.Info(manifestgenerator.Version, logPath)
	log.Info("Started tssegmenter", logPath)

	mg := manifestgenerator.New(log, *isCreatingChunks, *baseOutPath, *chunkBaseFilename, *targetSegmentDurS, *videoPID, *audioPID, manifestgenerator.ManifestTypes(*manifestTypeInt), *liveWindowSize, *lhlsAdvancedChunks)

	// Reader
	r := bufio.NewReader(os.Stdin)

	// Buffer
	buf := make([]byte, 0, readBufferSize)

	for {
		n, err := r.Read(buf[:cap(buf)])
		if n == 0 && err == io.EOF {
			// Detected EOF
			// Closing
			log.Info("Closing process detected EOF")
			mg.Close()

			break
		}

		if err != nil && err != io.EOF {
			// Error reading pipe
			log.Fatal(err, logPath)
			os.Exit(1)
		}

		// process buf
		log.Debug("Sent to process: ", n, " bytes")
		mg.AddData(buf[:n])
	}

	log.Info("Exit because detected EOF in the input pipe")

	os.Exit(0)
}
