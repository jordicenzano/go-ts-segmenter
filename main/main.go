package main

import (
	"flag"

	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator"
	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator/mediachunk"
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
	baseOutPath        = flag.String("p", "./results", "Output path")
	chunkBaseFilename  = flag.String("f", "chunk_", "Chunks base filename")
	targetSegmentDurS  = flag.Float64("t", 4.0, "Chunk duration in seconds")
	liveWindowSize     = flag.Int("w", 3, "Live window size in chunks")
	lhlsAdvancedChunks = flag.Int("l", 0, "LHLS advanced chunks")
	manifestTypeInt    = flag.Int("m", int(manifestgenerator.LiveWindow), "Manifest to generate (0- Vod, 1- Live event, 2- Live sliding window")
	autoPID            = flag.Bool("apids", true, "Enable auto PID detection, if true no need to pass vpid and apid")
	videoPID           = flag.Int("vpid", -1, "Video PID to parse")
	audioPID           = flag.Int("apid", -1, "Audio PID to parse")
	chunkInitType      = flag.Int("m", int(manifestgenerator.ChunkInitStart), "Indicates where to put the init data PAT and PMT packets (0- No ini data, 1- Init segment, 2- At the begining of each chunk")
	destinationType    = flag.Int("d", int(mediachunk.OutputModeFile), "Indicates where the destination (0- No output, 1- File + flag indicator)")
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

	if *autoPID == false && manifestgenerator.ChunkInitTypes(*chunkInitType) != manifestgenerator.ChunkNoIni {
		log.Error("Manual PID mode and Chunk No ini data are not compatible")
		os.Exit(1)
	}

	// Creating output dir if does not exists
	if mediachunk.OutputTypes(*destinationType) == mediachunk.OutputModeFile {
		os.MkdirAll(*baseOutPath, 0744)
	}

	mg := manifestgenerator.New(log,
		mediachunk.OutputTypes(*destinationType),
		*baseOutPath,
		*chunkBaseFilename,
		*targetSegmentDurS,
		manifestgenerator.ChunkInitTypes(*chunkInitType),
		*autoPID,
		-1,
		-1,
		manifestgenerator.ManifestTypes(*manifestTypeInt),
		*liveWindowSize,
		*lhlsAdvancedChunks,
	)

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
