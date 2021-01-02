#!/usr/bin/env bash

if [ $# -lt 1 ]; then
	echo "Use ./transmuxed-file-to-s3.sh FILE S3Bucket [S3Region] [StreamID] [Loop]\n"
    echo "S3Bucket: S3 bucket to use"
    echo "S3Region: S3 region to use (Default: us-east-1)"
    echo "StreamID: Stream name (Default: YYYYMMDDHHMMSS)"
    echo "Loop: Use file loop (Default 0)"
    echo "Example: ./transmuxed-file-to-s3.sh ~/test.mp4 testBucket us-east-1 streamKey 1"
    exit 1
fi

# Source file
SRC_FILE=$1

# S3 Data
S3_BUCKET=$2
S3_REGION="${3:-"us-east-1"}"

# ObjKeyPath
STREAM_ID_DEF=`date '+%Y%m%d%H%M%S'`
STREAM_ID="${4:-"$STREAM_ID_DEF"}"
DST_PATH="ingest/${STREAM_ID}"

# Set loop command
IS_LOOP="${5:-"0"}"
LOOP_CMD=""
if [ "$IS_LOOP" -eq "1" ]; then
    LOOP_CMD="-stream_loop -1"
fi

echo "Waiting for stream in: ${DST_PATH}"
echo "Using s3 upload path: ${DST_PATH}"

# Starts segmenter 
../bin/go-ts-segmenter -inputType 2 -targetDur 2 -manifestDestinationType 0 -s3Bucket $S3_BUCKET -s3Region $S3_REGION -mediaDestinationType 4 -dstPath $DST_PATH -chunksBaseFilename source_ &
PID_SRC=$!
echo "Started go-ts-segmenter for source as PID $PID_SRC"

# Start RTMP listener / transmuxer
ffmpeg -hide_banner -y \
-re $LOOP_CMD -i $SRC_FILE \
-c:v copy -c:a copy \
-f mpegts "tcp://localhost:2002"

# Destination
echo "You should be able to s3 the files in this S3 bucket $S3_BUCKET"
