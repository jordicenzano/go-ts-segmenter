#!/usr/bin/env bash

if [ $# -lt 1 ]; then
	echo "Use ./transmuxed-file-to-http.sh FILEin URLhost [streamID] [Loop]\n"
    echo "FILEin: Input file"
    echo "URLhost: Host-port to send the chunks via HTTP (default \"localhost:9094\""
    echo "streamID: StreamID to use, it is also the path for HTTP requests (default \"DATE WITH SECONDS\")"
    echo "Loop: Use file loop (Default 0)"
    echo "Example: ./transmuxed-file-to-http.sh ~/test.mp4 http://localhost:9094/ID1 1"
    exit 1
fi

# Source file
SRC_FILE=$1

# Dest host
DST_HOST="${2:-"localhost:9094"}"

# Output path (StreamID)
STREAM_ID_DEF=`date '+%Y%m%d%H%M%S'`
STREAM_ID="${3:-"$STREAM_ID_DEF"}"
DST_PATH="ingest/$STREAM_ID"

# Set loop command
IS_LOOP="${3:-"0"}"
LOOP_CMD=""
if [ "$IS_LOOP" -eq "1" ]; then
    LOOP_CMD="-stream_loop -1"
fi

echo "Output to: ${DST_PATH}"

# Starts segmenter 
../bin/go-ts-segmenter -inputType 2 -targetDur 4 -manifestDestinationType 0 -mediaDestinationType 3 -host $DST_HOST -dstPath $DST_PATH -chunksBaseFilename source_ &
PID_SRC=$!
echo "Started go-ts-segmenter for source as PID $PID_SRC"

# Allow server to start
sleep 2

# Start RTMP listener / transmuxer
ffmpeg -hide_banner -y \
-re $LOOP_CMD -i $SRC_FILE \
-c:v copy -c:a copy \
-f mpegts "tcp://localhost:2002"
