#!/usr/bin/env bash

BASE_DIR="../results/multirendition"

# Clean up
echo "Restarting ${BASE_DIR} directory"
rm -rf $BASE_DIR/*
mkdir -p $BASE_DIR

# Create master playlist (this should be created after 1st chunk is uploaded)
echo "Creating master playlist manifest (playlist.m3u8)"
echo "#EXTM3U" > $BASE_DIR/playlist.m3u8
echo "#EXT-X-VERSION:3" >> $BASE_DIR/playlist.m3u8
echo "#EXT-X-STREAM-INF:BANDWIDTH=996000,RESOLUTION=854x480" >> $BASE_DIR/playlist.m3u8
echo "480p.m3u8" >> $BASE_DIR/playlist.m3u8
echo "#EXT-X-STREAM-INF:BANDWIDTH=548000,RESOLUTION=640x360" >> $BASE_DIR/playlist.m3u8
echo "360p.m3u8" >> $BASE_DIR/playlist.m3u8

# Upload master playlist
curl http://localhost:9094/results/playlist.m3u8 --upload-file $BASE_DIR/playlist.m3u8

# Select font path based in OS
if [[ "$OSTYPE" == "linux-gnu" ]]; then
    FONT_PATH='/usr/share/fonts/dejavu/DejaVuSans-Bold.ttf'
elif [[ "$OSTYPE" == "darwin"* ]]; then
    FONT_PATH='/Library/Fonts/Arial.ttf'
fi

# Creates pipes
mkfifo $BASE_DIR/fifo-480p
mkfifo $BASE_DIR/fifo-360p

# Creates consumers
cat "$BASE_DIR/fifo-480p" | ../bin/manifest-generator -lf ../logs/segmenter480p.log -l 3 -d 2 -f 480p_ -cf 480p.m3u8 &
PID_480p=$!
echo "Started manifest-generator for 480p as PID $PID_480p"
cat "$BASE_DIR/fifo-360p" | ../bin/manifest-generator -lf ../logs/segmenter360p.log -l 3 -d 2 -f 360p_ -cf 360p.m3u8 &
PID_360p=$!
echo "Started manifest-generator for 360p as PID $PID_360p"

# Start test signal
ffmpeg -hide_banner -y \
-f lavfi -re -i smptebars=duration=6000:size=320x200:rate=30 \
-f lavfi -i sine=frequency=1000:duration=6000:sample_rate=48000 -pix_fmt yuv420p \
-vf scale=854x480 \
-vf "drawtext=fontfile=$FONT_PATH: text=\'RENDITION 480p - Local time %{localtime\: %Y\/%m\/%d %H.%M.%S} (%{n})\': x=0: y=0: fontsize=10: fontcolor=pink: box=1: boxcolor=0x00000099" \
-c:v libx264 -b:v 900k -g 60 -profile:v baseline -preset veryfast \
-c:a aac -b:a 48k \
-f mpegts "$BASE_DIR/fifo-480p" \
-vf scale=640x360 \
-vf "drawtext=fontfile=$FONT_PATH: text=\'RENDITION 360p - Local time %{localtime\: %Y\/%m\/%d %H.%M.%S} (%{n})\': x=0: y=0: fontsize=10: fontcolor=pink: box=1: boxcolor=0x00000099" \
-c:v libx264 -b:v 500k -g 60 -profile:v baseline -preset veryfast \
-c:a aac -b:a 48k \
-f mpegts "$BASE_DIR/fifo-360p"

# Clean up: Stop processes
# If the input stream stops the segmenter processes exists themselves
# kill $PID_480p
# kill $PID_360p
