#!/usr/bin/env bash

if [ $# -lt 2 ]; then
	echo "Use ./transmuxed-rtmp-to-s3.sh S3BUCKET StreamID [RTMPApp][RTMPPort]\n"
    echo "RTMPPort: RTMP stream name (example: 20201220101213)"
    echo "RTMPPort: RTMP app name (default: \"ingest\")"
    echo "RTMPPort: RTMP local port (default: 1935)"
    echo "Example: ./transmuxed-rtmp-to-s3.sh testBucket 20201220101213 live 1935"
    exit 1
fi

# S3 Bucket
S3_BUCKET=$1

# RTMP settings
STREAM_ID=$2
RTMP_APP="${3:-"ingest"}"
RTMP_PORT="${4:-"1935"}"

# ObjKeyPath
DST_PATH="${RTMP_APP}/${STREAM_ID}"

echo "Waiting for stream in: ${DST_PATH}"
echo "Using s3 upload path: ${DST_PATH}"

# Starts segmenter 
../bin/go-ts-segmenter -inputType 2 -manifestDestinationType 0 -s3Bucket $S3_BUCKET -mediaDestinationType 4 -dstPath $DST_PATH -chunksBaseFilename source_ &
PID_SRC=$!
echo "Started go-ts-segmenter for source as PID $PID_SRC"

# Start RTMP listener / transmuxer
ffmpeg -hide_banner -y \
-listen 1 -i "rtmp://0.0.0.0:$RTMP_PORT/$RTMP_APP/$STREAM_ID" \
-c:v copy -c:a copy \
-f mpegts "tcp://localhost:2002"

# Destination
echo "You should be able to s3 the files in this S3 bucket $S3_BUCKET"
