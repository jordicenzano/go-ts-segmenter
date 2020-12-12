package s3uploader

import (
	"strconv"
	"testing"
	"time"
)

func getDateTimeStr() string {
	t := time.Now()
	return strconv.Itoa(t.Year()) + strconv.Itoa(int(t.Month())) + strconv.Itoa(t.Day()) + strconv.Itoa(t.Hour()) + strconv.Itoa(t.Minute()) + strconv.Itoa(t.Second())
}

// Do not run this test, it uses real S3 and local default config
func TestUploadLocalFile(t *testing.T) {
	testFilePath := "../../fixture/testSmall.ts"
	headerName := "hName"
	headerValue := "hValue"
	UploadFilePath := "input/testID/testSmall-" + getDateTimeStr() + ".ts"

	// Used computer default creds
	awsCreds := AWSLocalCreds{Valid: false}
	// Upload to test bucket
	up := New(nil, "live-dist-test", "us-east-1", 10000, false, awsCreds)

	// Test metadata
	h := map[string]string{headerName: headerValue, "Content-Type": "video/MP2T"}

	ret := up.UploadLocalFile(testFilePath, UploadFilePath, h)
	if ret != nil {
		t.Error("Error uploading localfile. Err ", ret)
	}
}
