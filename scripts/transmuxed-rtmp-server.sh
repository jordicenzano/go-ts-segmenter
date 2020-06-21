#!/usr/bin/env bash

if [ $# -lt 1 ]; then
	echo "Use ./transmuxed-rtmp-server.sh test/live [RTMPPort] [RTMPApp] [RTMPStream] [HLSOutHostPort]"
    echo "test/live: Generates test signal, no need for RTMP source"
    echo "RTMPPort: RTMP local port (default: 1935)"
    echo "RTMPPort: RTMP app name (default: \"live\")"
    echo "RTMPPort: RTMP stream name (default: \"stream\")"
    echo "HLSOutHostPort: Host and to send HLS data (default: \"localhost:9094\")"
    echo "Example: ./transmuxed-rtmp-server.sh live \"localhost:9094\" 1935 \"live\" \"stream\""
    exit 1
fi

MODE="${1}"
RTMP_PORT="${2:-"1935"}"
RTMP_APP="${3:-"live"}"
RTMP_STREAM="${4:-"stream"}"
HOST_DST="${5:-"localhost:9094"}"

PATH_NAME="srrtmp"
STREAM_NAME="source"
BASE_DIR="../results/${PATH_NAME}"

# Clean up
echo "Restarting ${BASE_DIR} directory"
rm -rf $BASE_DIR/*
mkdir -p $BASE_DIR

# Create master playlist (this should be created after 1st chunk is uploaded)
# Assuming source is 1280x720@6Mbps
echo "Creating master playlist manifest (playlist.m3u8)"
echo "#EXTM3U" > $BASE_DIR/playlist.m3u8
echo "#EXT-X-VERSION:3" >> $BASE_DIR/playlist.m3u8
echo "#EXT-X-STREAM-INF:BANDWIDTH=6000000,RESOLUTION=1280x720" >> $BASE_DIR/playlist.m3u8
echo "$STREAM_NAME.m3u8" >> $BASE_DIR/playlist.m3u8

# Upload master playlist
curl "http://${HOST_DST}/${PATH_NAME}/playlist.m3u8" -H "Content-Type: application/vnd.apple.mpegurl" --upload-file $BASE_DIR/playlist.m3u8

# Select font path based in OS
if [[ "$OSTYPE" == "linux-gnu" ]]; then
    FONT_PATH='/usr/share/fonts/dejavu/DejaVuSans-Bold.ttf'
elif [[ "$OSTYPE" == "darwin"* ]]; then
    FONT_PATH='/Library/Fonts/Arial.ttf'
fi

# Creates pipe
FIFO_FILENAME="fifo-$STREAM_NAME"
mkfifo $BASE_DIR/$FIFO_FILENAME

# Creates hls producer
cat "$BASE_DIR/$FIFO_FILENAME" | ../bin/manifest-generator -lf ../logs/segmenterSource.log -p ${PATH_NAME} -manifestDestinationType 2 -mediaDestinationType 2 -t 1 -l 3 -f ${STREAM_NAME}_ -cf ${STREAM_NAME}.m3u8 &
PID_SOURCE=$!
echo "Started manifest-generator for $STREAM_NAME as PID $PID_SOURCE"

if [[ "$MODE" == "test" ]]; then
    # Start test signal
    # GOP size = 30f @ 30 fps = 1s
    ffmpeg -hide_banner -y \
    -f lavfi -re -i smptebars=duration=36000:size=1280x720:rate=30 \
    -f lavfi -i sine=frequency=1000:duration=36000:sample_rate=48000 -pix_fmt yuv420p \
    -vf scale=1280x720 \
    -vf "drawtext=fontfile=$FONT_PATH:text=\'RENDITION 720p - Local time %{localtime\: %Y\/%m\/%d %H.%M.%S} (%{n})\':x=10:y=350:fontsize=30:fontcolor=pink:box=1:boxcolor=0x00000099" \
    -c:v libx264 -tune zerolatency -b:v 6000k -g 30 -preset ultrafast \
    -c:a aac -b:a 48k \
    -f mpegts "$BASE_DIR/$FIFO_FILENAME"
else
    # Start transmuxer
    ffmpeg -hide_banner -y \
    -listen 1 -i "rtmp://0.0.0.0:$RTMP_PORT/$RTMP_APP/$RTMP_STREAM" \
    -c:v copy -c:a copy \
    -f mpegts "$BASE_DIR/$FIFO_FILENAME"
fi

# Clean up: Stop processes
# If the input stream stops the segmenter processes exists themselves
# kill $PID_SOURCE
