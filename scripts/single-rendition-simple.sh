#!/usr/bin/env bash

HOST_DST="localhost:9094"

TEXT="SOURCE-"

BASE_DIR="../results/singlerendition"

UPLOAD_PATH="results"

# Clean up
echo "Restarting ${BASE_DIR} directory"
rm -rf $BASE_DIR/*
mkdir -p $BASE_DIR

# Create master playlist (this should be created after 1st chunk is uploaded)
echo "Creating master playlist manifest (playlist.m3u8)"
echo "#EXTM3U" > $BASE_DIR/playlist.m3u8
echo "#EXT-X-VERSION:3" >> $BASE_DIR/playlist.m3u8
echo "#EXT-X-STREAM-INF:BANDWIDTH=6144000,RESOLUTION=1280x720" >> $BASE_DIR/playlist.m3u8
echo "720p.m3u8" >> $BASE_DIR/playlist.m3u8

# Upload master playlist
curl "http://$HOST_DST/$UPLOAD_PATH/playlist.m3u8" --upload-file $BASE_DIR/playlist.m3u8

# Select font path based in OS
# TODO: Probably (depending on the distrubuition) for linux you will need to find the right path
if [[ "$OSTYPE" == "linux-gnu" ]]; then
    FONT_PATH='/usr/share/fonts/Hack-Regular.ttf'
elif [[ "$OSTYPE" == "darwin"* ]]; then
    FONT_PATH='/Library/Fonts/Arial.ttf'
fi

# Creates pipes
mkfifo $BASE_DIR/fifo-720p

# Creates consumers
cat "$BASE_DIR/fifo-720p" | ../bin/go-ts-segmenter -p $UPLOAD_PATH -lf ../logs/segmenter720p.log -host $HOST_DST -manifestDestinationType 2 -mediaDestinationType 2 -f 720p_ -cf 720p.m3u8 &
PID_720p=$!
echo "Started go-ts-segmenter for 720p as PID $PID_720p"

# Start test signal
ffmpeg -hide_banner -y \
-f lavfi -re -i smptebars=size=1280x720:rate=30 \
-f lavfi -i sine=frequency=1000:sample_rate=48000 -pix_fmt yuv420p \
-vf "drawtext=fontfile=$FONT_PATH: text=\'${TEXT} 720p - Local time %{localtime\: %Y\/%m\/%d %H.%M.%S} (%{n})\': x=100: y=50: fontsize=30: fontcolor=pink: box=1: boxcolor=0x00000099" \
-c:v libx264 -b:v 6000k -g 60 -profile:v baseline -preset veryfast \
-c:a aac -b:a 48k \
-f mpegts "$BASE_DIR/fifo-720p"

# Clean up: Stop process
# If the input stream stops the segmenter process exists themselves
# kill $PID_720p
