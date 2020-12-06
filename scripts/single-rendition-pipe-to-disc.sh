#!/usr/bin/env bash

if [ $# -lt 1 ]; then
	echo "Use ./single-rendition-pipe-to-disc.sh [path] [TEXT-TO-PREFIX-TO-PATH]\n"
    echo "Example: ./single-rendition-pipe-to-disc.sh \"../results\" \"pipe-disc\""
	# exit 1
fi

# Base path
DST_BASE_PATH="${1:-"../results"}"

# Append to path
PATH_PREFIX="${2:-"pipe-disc"}"

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
echo "#EXT-X-STREAM-INF:BANDWIDTH=6144000,RESOLUTION=1280x720" >> $DST_PATH/playlist.m3u8
echo "720p.m3u8" >> $DST_PATH/playlist.m3u8

# Creates pipes
rm $DST_PATH/fifo-720p
mkfifo $DST_PATH/fifo-720p

# Starts segmenter 
cat $DST_PATH/fifo-720p | ../bin/go-ts-segmenter -dstPath ${DST_PATH} -chunksBaseFilename 720p_ -chunklistFilename 720p.m3u8 &
PID_720p=$!
echo "Started go-ts-segmenter for 720p as PID $PID_720p"

# Select font path based in OS
# TODO: Probably (depending on the distribuition) for linux you will need to find the right path
if [[ "$OSTYPE" == "linux-gnu" ]]; then
    FONT_PATH='/usr/share/fonts/Hack-Regular.ttf'
elif [[ "$OSTYPE" == "darwin"* ]]; then
    FONT_PATH='/Library/Fonts/Arial.ttf'
fi

# Start test signal
ffmpeg -hide_banner -y \
-f lavfi -re -i smptebars=size=1280x720:rate=30 \
-f lavfi -i sine=frequency=1000:sample_rate=48000 -pix_fmt yuv420p \
-vf "drawtext=fontfile=$FONT_PATH: text=\'${TEXT} 720p - Local time %{localtime\: %Y\/%m\/%d %H.%M.%S} (%{n})\': x=100: y=50: fontsize=30: fontcolor=pink: box=1: boxcolor=0x00000099" \
-c:v libx264 -b:v 6000k -g 60 -profile:v baseline -preset veryfast \
-c:a aac -b:a 48k \
-f mpegts "$DST_PATH/fifo-720p"

# Playback
ABS_DST_PATH=`realpath ${DST_PATH}`
echo "You should be able to play this HLS manifest $ABS_DST_PATH/playlist.m3u8. Example: ffplay $ABS_DST_PATH/playlist.m3u8"
