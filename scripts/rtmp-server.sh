#!/usr/bin/env bash

BASE_DIR="../results/rtmpsvr"

TEXT="SOURCE-RTMP-"

# Clean up
echo "Restarting ${BASE_DIR} directory"
rm -rf $BASE_DIR/*
mkdir -p $BASE_DIR

# Create master playlist (this should be created after 1st chunk is uploaded)
# It is assuming 1920x1080 input @6Mbps
echo "Creating master playlist manifest (playlist.m3u8)"
echo "#EXTM3U" > $BASE_DIR/playlist.m3u8
echo "#EXT-X-VERSION:3" >> $BASE_DIR/playlist.m3u8
echo "#EXT-X-STREAM-INF:BANDWIDTH=6144000,RESOLUTION=1920x1080" >> $BASE_DIR/playlist.m3u8
echo "1080p.m3u8" >> $BASE_DIR/playlist.m3u8

# Upload master playlist
curl http://localhost:9094/results/playlist.m3u8 --upload-file $BASE_DIR/playlist.m3u8

# Creates pipes
mkfifo $BASE_DIR/fifo-rtmp

# Creates consumers
cat "$BASE_DIR/fifo-rtmp" | ../bin/manifest-generator -lf ../logs/segmenter720p.log -d 2 -f source_ -cf source.m3u8 &
PID_SOURCE=$!
echo "Started manifest-generator for RTMP stream as PID $PID_SOURCE"

# Start test signal
ffmpeg -hide_banner -y \
-listen 1 -i "rtmp://0.0.0.0:1935/live/stream" \
-c:v copy \
-c:a copy \
-f mpegts "$BASE_DIR/fifo-rtmp"

# Clean up: Stop process
# If the input stream stops the segmenter process exists themselves
# kill $PID_SOURCE
