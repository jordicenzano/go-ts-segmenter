package main

import (
	"flag"
	"net/http"

	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator"
	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator/hls"
	"github.com/jordicenzano/go-ts-segmenter/manifestgenerator/mediachunk"
	"github.com/sirupsen/logrus"

	"bufio"
	"fmt"
	"io"
	"os"
)

const (
	readBufferSize = 128
)

var (
	verbose                 = flag.Bool("v", false, "enable to get verbose logging")
	baseOutPath             = flag.String("p", "./results", "Output path")
	chunkBaseFilename       = flag.String("f", "chunk_", "Chunks base filename")
	chunkListFilename       = flag.String("cf", "chunklist.m3u8", "Chunklist filename")
	targetSegmentDurS       = flag.Float64("t", 4.0, "Target chunk duration in seconds")
	liveWindowSize          = flag.Int("w", 3, "Live window size in chunks")
	lhlsAdvancedChunks      = flag.Int("l", 0, "If > 0 activates LHLS, and it indicates the number of advanced chunks to create")
	manifestTypeInt         = flag.Int("m", int(hls.LiveWindow), "Manifest to generate (0- Vod, 1- Live event, 2- Live sliding window")
	autoPID                 = flag.Bool("apids", true, "Enable auto PID detection, if true no need to pass vpid and apid")
	videoPID                = flag.Int("vpid", -1, "Video PID to parse")
	audioPID                = flag.Int("apid", -1, "Audio PID to parse")
	chunkInitType           = flag.Int("i", int(manifestgenerator.ChunkInitStart), "Indicates where to put the init data PAT and PMT packets (0- No ini data, 1- Init segment, 2- At the begining of each chunk")
	mediaDestinationType    = flag.Int("mediaDestinationType", 1, "Indicates where the destination (0- No output, 1- File + flag indicator, 2- HTTP chunked transfer, 3- HTTP regular)")
	manifestDestinationType = flag.Int("manifestDestinationType", 1, "Indicates where the destination (0- No output, 1- File + flag indicator, 2- HTTP)")
	httpScheme              = flag.String("protocol", "http", "HTTP Scheme (http, https)")
	httpHost                = flag.String("host", "localhost:9094", "HTTP Host")
	logPath                 = flag.String("lf", "./logs/segmenter.log", "Logs file")
)

func main() {
	flag.Parse()

	var log = configureLogger(*verbose, *logPath)

	log.Info(manifestgenerator.Version, logPath)
	log.Info("Started tssegmenter", logPath)

	if *autoPID == false && manifestgenerator.ChunkInitTypes(*chunkInitType) != manifestgenerator.ChunkNoIni {
		log.Error("Manual PID mode and Chunk No ini data are not compatible")
		os.Exit(1)
	}

	chunkOutputType := mediachunk.OutputTypes(*mediaDestinationType)
	hlsOutputType := hls.OutputTypes(*manifestDestinationType)

	// Creating output dir if does not exists
	if chunkOutputType == mediachunk.ChunkOutputModeFile || hlsOutputType == hls.HlsOutputModeFile {
		os.MkdirAll(*baseOutPath, 0744)
	}

	tr := http.DefaultTransport
	client := http.Client{
		Transport: tr,
		Timeout:   0,
	}

	mg := manifestgenerator.New(log,
		chunkOutputType,
		hlsOutputType,
		*baseOutPath,
		*chunkBaseFilename,
		*chunkListFilename,
		*targetSegmentDurS,
		manifestgenerator.ChunkInitTypes(*chunkInitType),
		*autoPID,
		-1,
		-1,
		hls.ManifestTypes(*manifestTypeInt),
		*liveWindowSize,
		*lhlsAdvancedChunks,
		&client,
		*httpScheme,
		*httpHost,
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

func configureLogger(verbose bool, logPath string) *logrus.Logger {
	var log = logrus.New()
	if verbose {
		log.SetLevel(logrus.DebugLevel)
	}

	formatter := new(logrus.JSONFormatter)
	formatter.TimestampFormat = "01-01-2001 13:00:00"

	log.SetFormatter(formatter)
	log.SetFormatter(&logrus.JSONFormatter{})

	f, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf(fmt.Sprintf("Unable to open log file at: %s, error: %v", logPath, err))
		os.Exit(-1)
	}

	// TESTING - write to both file and stdout for now
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)

	return log
}
