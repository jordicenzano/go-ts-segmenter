#!/usr/bin/env bash

if [ $# -lt 1 ]; then
	echo "Use ./multi-rendition-pipe-to-http.sh [host][path] [TEXT-TO-PREFIX-TO-PATH]\n"
    echo "Example: ./multi-rendition-pipe-to-http.sh \"localhost:9094\" \"../results\" \"tcp-disc\""
	# exit 1
fi

# Host port
HOST_DST="${1:-"localhost:9094"}"

# Base path
DST_BASE_PATH="${2:-"../results"}"

# Append to path
PATH_PREFIX="${3:-"pipe-http"}"

DST_PATH="${DST_BASE_PATH}/${PATH_PREFIX}"

# Overlay base text
TEXT="SOURCE-"

# Create destination dir
echo "Creating ${DST_PATH} directory (if necessary)"
mkdir -p $DST_PATH

# Make sure destination dir is empty
if [ -z "$(ls -A ${DST_PATH})" ]; then
   echo "The directory ${DST_PATH} is empty, ready for the test"
else
   echo "Stopping directory ${DST_PATH} NOT empty!!!!"
   exit 1
fi

# Create master playlist (this should be created after 1st chunk is uploaded)
echo "Creating master playlist manifest (playlist.m3u8)"
echo "#EXTM3U" > $DST_PATH/playlist.m3u8
echo "#EXT-X-VERSION:3" >> $DST_PATH/playlist.m3u8
echo "#EXT-X-STREAM-INF:BANDWIDTH=996000,RESOLUTION=854x480" >> $DST_PATH/playlist.m3u8
echo "480p.m3u8" >> $DST_PATH/playlist.m3u8
echo "#EXT-X-STREAM-INF:BANDWIDTH=548000,RESOLUTION=640x360" >> $DST_PATH/playlist.m3u8
echo "360p.m3u8" >> $DST_PATH/playlist.m3u8

# Upload master playlist
curl "http://${HOST_DST}/${PATH_PREFIX}/playlist.m3u8" --upload-file $DST_PATH/playlist.m3u8

# Creates pipes
rm $DST_PATH/fifo-480p
mkfifo $DST_PATH/fifo-480p
rm $DST_PATH/fifo-360p
mkfifo $DST_PATH/fifo-360p

# Starts segmenters
# Creates consumers
cat "$DST_PATH/fifo-480p" | ../bin/go-ts-segmenter -dstPath $PATH_PREFIX -lhls 3 -host $HOST_DST -manifestDestinationType 2 -mediaDestinationType 2 -chunksBaseFilename 480p_ -chunklistFilename 480p.m3u8 &
PID_480p=$!
echo "Started go-ts-segmenter for 480p as PID $PID_480p"
cat "$DST_PATH/fifo-360p" | ../bin/go-ts-segmenter -dstPath $PATH_PREFIX -lhls 3 -host $HOST_DST -manifestDestinationType 2 -mediaDestinationType 2 -chunksBaseFilename 360p_ -chunklistFilename 360p.m3u8 &
PID_360p=$!
echo "Started go-ts-segmenter for 360p as PID $PID_360p"

# Select font path based in OS
# TODO: Probably (depending on the distribuition) for linux you will need to find the right path
if [[ "$OSTYPE" == "linux-gnu" ]]; then
    FONT_PATH='/usr/share/fonts/Hack-Regular.ttf'
elif [[ "$OSTYPE" == "darwin"* ]]; then
    FONT_PATH='/Library/Fonts/Arial.ttf'
fi

# Playback
PLAYBACK_URL="http://${HOST_DST}/${PATH_PREFIX}/playlist.m3u8"
echo "You should be able to play this HLS manifest on this URL: ${PLAYBACK_URL}. Example: ffplay ${PLAYBACK_URL}"

# Start test signal
ffmpeg -hide_banner -y \
-f lavfi -re -i smptebars=duration=6000:size=320x200:rate=30 \
-f lavfi -i sine=frequency=1000:duration=6000:sample_rate=48000 -pix_fmt yuv420p \
-s 854x480 -vf "drawtext=fontfile=$FONT_PATH: text=\'RENDITION 480p - Local time %{localtime\: %Y\/%m\/%d %H.%M.%S} (%{n})\': x=0: y=0: fontsize=10: fontcolor=pink: box=1: boxcolor=0x00000099" \
-c:v libx264 -b:v 900k -g 60 -profile:v baseline -preset veryfast \
-c:a aac -b:a 48k \
-f mpegts "$DST_PATH/fifo-480p" \
-s 640x360 -vf "drawtext=fontfile=$FONT_PATH: text=\'RENDITION 360p - Local time %{localtime\: %Y\/%m\/%d %H.%M.%S} (%{n})\': x=0: y=0: fontsize=10: fontcolor=pink: box=1: boxcolor=0x00000099" \
-c:v libx264 -b:v 500k -g 60 -profile:v baseline -preset veryfast \
-c:a aac -b:a 48k \
-f mpegts "$DST_PATH/fifo-360p"
