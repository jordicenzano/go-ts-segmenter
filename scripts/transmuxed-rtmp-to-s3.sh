#!/usr/bin/env bash

if [ $# -lt 3 ]; then
	echo "Use ./transmuxed-rtmp-to-s3.sh S3Bucket S3Region StreamID [RTMPApp] [RTMPPort]\n"
    echo "StreamID: Also used as RTMP stream name (example: 20201220101213)"
    echo "RTMPApp: RTMP app name (default: \"ingest\")"
    echo "RTMPPort: RTMP local port (default: 1935)"
    echo "Example: ./transmuxed-rtmp-to-s3.sh testBucket us-east-1 20201220101213 live 1935"
    exit 1
fi

# S3 Data
S3_BUCKET=$1
S3_REGION=$2

# RTMP settings
STREAM_ID=$3
RTMP_APP="${4:-"ingest"}"
RTMP_PORT="${5:-"1935"}"

# ObjKeyPath
DST_PATH="${RTMP_APP}/${STREAM_ID}"

echo "Waiting for stream in: ${DST_PATH}"
echo "Using s3 upload path: ${DST_PATH}"

# Starts segmenter 
../bin/go-ts-segmenter -inputType 2 -manifestDestinationType 0 -s3Bucket $S3_BUCKET -s3Region $S3_REGION -mediaDestinationType 4 -dstPath $DST_PATH -chunksBaseFilename source_ &
PID_SRC=$!
echo "Started go-ts-segmenter for source as PID $PID_SRC"

# Start RTMP listener / transmuxer
ffmpeg -hide_banner -y \
-listen 1 -i "rtmp://0.0.0.0:$RTMP_PORT/$RTMP_APP/$STREAM_ID" \
-c:v copy -c:a copy \
-f mpegts "tcp://localhost:2002"

# Destination
echo "You should be able to s3 the files in this S3 bucket $S3_BUCKET"
