#!/usr/bin/env bash

if [ $# -lt 1 ]; then
	echo "Use ./transmuxed-file-to-s3.sh FILE S3Bucket [S3Region] [streamID]\n"
    echo "streamID: RTMP stream name (example: 20201220101213)"
    echo "Example: ./transmuxed-file-to-s3.sh ~/test.mp4 testBucket us-east-1 streamKey"
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

echo "Waiting for stream in: ${DST_PATH}"
echo "Using s3 upload path: ${DST_PATH}"

# Starts segmenter 
../bin/go-ts-segmenter -inputType 2 -targetDur 2 -manifestDestinationType 0 -s3Bucket $S3_BUCKET -s3Region $S3_REGION -mediaDestinationType 4 -dstPath $DST_PATH -chunksBaseFilename source_ &
PID_SRC=$!
echo "Started go-ts-segmenter for source as PID $PID_SRC"

# Start RTMP listener / transmuxer
ffmpeg -hide_banner -y \
-re -i $SRC_FILE \
-c:v copy -c:a copy \
-f mpegts "tcp://localhost:2002"

# Destination
echo "You should be able to s3 the files in this S3 bucket $S3_BUCKET"
