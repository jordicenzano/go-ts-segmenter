#!/usr/bin/env bash

if [ $# -lt 1 ]; then
	echo "Use ./single-rendition-media-tcp-to-s3.sh S3BUCKET [TEXT-TO-PREFIX-TO-PATH]\n"
    echo "Example: ./single-rendition-media-tcp-to-s3.sh "
	#exit 1
fi

# S3 Bucket
S3_BUCKET="${1:-"live-dist-test"}"

# Overlay base text
TEXT="SOURCE-"

# ObjKeyPath
STREAM_ID=`date '+%Y%m%d%H%M%S'`
DST_PATH="ingest/$STREAM_ID"

# Starts segmenter 
../bin/go-ts-segmenter -inputType 2 -manifestDestinationType 0 -s3Bucket $S3_BUCKET -mediaDestinationType 4 -dstPath $DST_PATH -chunksBaseFilename source_ &
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
-f mpegts "tcp://localhost:2002"

# Destination
echo "You should be able to s3 the files in this S3 bucket $S3_BUCKET"
