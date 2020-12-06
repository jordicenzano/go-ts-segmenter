package s3uploader

import (
	"crypto/rand"
	"fmt"
	"testing"
)

// Note - NOT RFC4122 compliant
func pseudoUUID() (uuid string) {

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return
}

// Do not run this test, it uses real S3 and local default config
func TestUploadLocalFile(t *testing.T) {
	testFilePath := "../../fixture/testSmall.ts"
	headerName := "hName"
	headerValue := "hValue"
	UploadFilePath := "test/testSmall-" + pseudoUUID() + ".ts"

	// Used computer default creds
	awsCreds := AWSLocalCreds{valid: false}
	// Upload to test bucket
	up := New(nil, "live-dist-test", "us-east-1", 10000, false, awsCreds)

	// Test metadata
	h := map[string]string{headerName: headerValue, "Content-Type": "video/MP2T"}

	ret := up.UploadLocalFile(testFilePath, UploadFilePath, h)
	if ret != nil {
		t.Error("Error uploading localfile. Err ", ret)
	}
}
