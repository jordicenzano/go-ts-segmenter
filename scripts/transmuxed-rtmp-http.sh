#!/usr/bin/env bash

if [ $# -lt 1 ]; then
	echo "Use ./transmuxed-rtmp-to-http.sh streamID [URLhost] [RTMPApp] [RTMPPort]\n"
    echo "streamID: StreamID to use, it is also the path for HTTP requests (default \"DATE WITH SECONDS\")"
    echo "URLhost: Host-port to send the chunks via HTTP (default \"localhost:9094\""
    echo "RTMPApp: RTMP app name (default: \"ingest\")"
    echo "RTMPPort: RTMP local port (default: 1935)"
    echo "Example: ./transmuxed-rtmp-to-http.sh localhost:9094 20201220101213 live 1935"
    exit 1
fi

# Output path (StreamID)
STREAM_ID=$1

# Dest host
DST_HOST="${2:-"localhost:9094"}"

# RTMP settings
RTMP_APP="${3:-"ingest"}"
RTMP_PORT="${4:-"1935"}"

# ObjKeyPath
DST_PATH="ingest/$STREAM_ID"

echo "Waiting for RTMP stream in: ${RTMP_APP}/${STREAM_ID}"

# Starts segmenter 
../bin/go-ts-segmenter -inputType 2 -targetDur 4 -manifestDestinationType 0 -mediaDestinationType 3 -host $DST_HOST -dstPath $DST_PATH -chunksBaseFilename source_ &
PID_SRC=$!
echo "Started go-ts-segmenter for source as PID $PID_SRC"

# Start RTMP listener / transmuxer
ffmpeg -hide_banner -y \
-listen 1 -i "rtmp://0.0.0.0:$RTMP_PORT/$RTMP_APP/$STREAM_ID" \
-c:v copy -c:a copy \
-f mpegts "tcp://localhost:2002"
