#!/usr/bin/env bash

if [ $# -lt 2 ]; then
	echo "Use ./single-rendition-file.sh host file [TEXT-TO-PREFIX-TO-PATH]\n"
    echo "Example: ./single-rendition-file.sh \"localhost:9094\" \"/media/test.mp4\""
	exit 1
fi

HOST_DST=$1
FILE_SRC=$2

BASE_DIR="../results/singlerendition"

# Clean up
echo "Restarting ${BASE_DIR} directory"
rm -rf $BASE_DIR/*
mkdir -p $BASE_DIR

# Append to random path
PATH_PREFIX=""
if [ $# -gt 2 ]; then
    PATH_PREFIX=$2
fi

# Generate random string
RANDOM_STR=`openssl rand -hex 8`
UPLOAD_PATH="${PATH_PREFIX}${RANDOM_STR}"
echo "Ramdom stream path: ${UPLOAD_PATH}"

# Creates pipes
mkfifo $BASE_DIR/fifo-source

# Creates consumers
cat "$BASE_DIR/fifo-source" | ../bin/manifest-generator -p $UPLOAD_PATH -lf ../logs/segmenter-source.log -host "$HOST_DST" -manifestDestinationType 0 -mediaDestinationType 3 -f source_ -cf "source.m3u8" &
PID_SOURCE=$!
echo "Started manifest-generator for source as PID $PID_SOURCE"

# Start test signal
ffmpeg -hide_banner -y \
-stream_loop -1 -re -i "$FILE_SRC" \
-c:v copy \
-c:a copy \
-f mpegts "$BASE_DIR/fifo-source"

# Clean up: Stop process
# If the input stream stops the segmenter process exists themselves
# kill $PID_SOURCE