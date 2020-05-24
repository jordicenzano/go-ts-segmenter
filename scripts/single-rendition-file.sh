#!/usr/bin/env bash

#!/usr/bin/env bash

BASE_DIR="../results/singlerendition"

# Clean up
echo "Restarting ${BASE_DIR} directory"
rm -rf $BASE_DIR/*
mkdir -p $BASE_DIR

# Generate random string
RANDOM_STR=`openssl rand -hex 8`
echo "Ramdom stream path: $RANDOM_STR"

# Creates pipes
mkfifo $BASE_DIR/fifo-source

# Creates consumers
cat "$BASE_DIR/fifo-source" | ../bin/manifest-generator -p $RANDOM_STR -lf ../logs/segmenter-source.log -host "hls-transocoder-public-v1-855763197.us-east-1.elb.amazonaws.com:8080" -manifestDestinationType 0 -mediaDestinationType 3 -f source_ -cf "source.m3u8" &
PID_SOURCE=$!
echo "Started manifest-generator for source as PID $PID_SOURCE"

# Start test signal
ffmpeg -hide_banner -y \
-stream_loop -1 -re -i "$1" \
-c:v copy \
-c:a copy \
-f mpegts "$BASE_DIR/fifo-source"

# Clean up: Stop process
# If the input stream stops the segmenter process exists themselves
# kill $PID_SOURCE