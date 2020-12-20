#!/usr/bin/env bash

if [ $# -lt 2 ]; then
	echo "Use ./transmuxed-rtmp-to-s3.sh S3Bucket S3Region [StreamID] [SRTPort]\n"
    echo "StreamID: Used to upload to S3 (default: YYYYMMDDHHMMSS)"
    echo "SRTPort: SRT local port (default: 1935)"
    echo "Example: ./transmuxed-srt-to-s3.sh testBucket us-east-1 20201220101213 1935"
    exit 1
fi

# S3 Data
S3_BUCKET=$1
S3_REGION=$2

# ObjKeyPath
STREAM_ID_DEF=`date '+%Y%m%d%H%M%S'`
STREAM_ID="${3:-"$STREAM_ID_DEF"}"

# SRT Data
SRT_PORT="${4:-"1935"}"

# ObjKeyPath
DST_PATH="ingest/${STREAM_ID}"

echo "Waiting for stream in: ${DST_PATH}"
echo "Using s3 upload path: ${DST_PATH}"

# Starts segmenter 
../bin/go-ts-segmenter -inputType 2 -targetDur 2 -manifestDestinationType 0 -s3Bucket $S3_BUCKET -s3Region $S3_REGION -mediaDestinationType 4 -dstPath $DST_PATH -chunksBaseFilename source_ &
PID_SRC=$!
echo "Started go-ts-segmenter for source as PID $PID_SRC"

# Start RTMP listener / transmuxer
ffmpeg -hide_banner -y \
-i "srt://0.0.0.0:$SRT_PORT?mode=listener" \
-c:v copy -c:a copy \
-f mpegts "tcp://localhost:2002"

# Destination
echo "You should be able to s3 the files in this S3 bucket $S3_BUCKET"
