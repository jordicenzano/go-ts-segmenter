#!/usr/bin/env bash

if [ $# -lt 2 ]; then
	echo "Use ./transmuxed-file-to-file.sh FILEin FILEout [Loop]\n"
    echo "FILEin: Input file"
    echo "FILEin: Output file path"
    echo "Loop: Use file loop (Default 0)"
    echo "Example: ./transmuxed-file-to-file.sh ~/test.mp4 ~/ouput 1"
    exit 1
fi

# Source file
SRC_FILE=$1

# S3 Data
DST_PATH=$2

# Set loop command
IS_LOOP="${5:-"0"}"
LOOP_CMD=""
if [ "$IS_LOOP" -eq "1" ]; then
    LOOP_CMD="-stream_loop -1"
fi

echo "Output to: ${DST_PATH}"

# Starts segmenter 
../bin/go-ts-segmenter -inputType 2 -targetDur 4 -manifestDestinationType 0 -mediaDestinationType 1 -dstPath $DST_PATH -chunksBaseFilename source_ &
PID_SRC=$!
echo "Started go-ts-segmenter for source as PID $PID_SRC"

# Start RTMP listener / transmuxer
ffmpeg -hide_banner -y \
-re $LOOP_CMD -i $SRC_FILE \
-c:v copy -c:a copy \
-f mpegts "tcp://localhost:2002"

# Destination
echo "You should be able to see the output segments in $DST_PATH"
