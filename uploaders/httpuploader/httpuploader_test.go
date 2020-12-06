package httpuploader

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
)

// TestMain will exec each test, one by one
func TestMain(m *testing.M) {
	// Do your stuff here
	os.Exit(m.Run())
}

func testBinary(a, b []byte) bool {

	// If one is nil, the other must also be nil.
	if (a == nil) != (b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestUploadLocalFile(t *testing.T) {
	testFilePath := "../../fixture/testSmall.ts"
	headerName := "hName"
	headerValue := "hValue"
	UploadFilePath := "test/testSmall.ts"

	serverHandleTest := func(rw http.ResponseWriter, req *http.Request) {
		// Check path
		if req.URL.Path != "/"+UploadFilePath {
			t.Errorf("Server received wrong path, got: %s, want: %s.", req.URL.String(), UploadFilePath)
		}

		// Check headers
		hVal := req.Header.Get(headerName)
		if hVal == "" {
			t.Errorf("Header not received, got: \"\", want: %s.", headerName)
		} else if hVal != headerValue {
			t.Errorf("Header value received is wrong, got: %s, want: %s.", hVal, headerValue)
		}

		// Check data
		buf, errReadReq := ioutil.ReadAll(req.Body)
		if errReadReq != nil {
			t.Error("Error reading the sent body. Err: ", errReadReq)
		}

		fExpected, errOpenLocalExpected := os.Open(testFilePath)
		if errOpenLocalExpected != nil {
			t.Error("Error opening the local expected result file. Err: ", errOpenLocalExpected)
		}
		defer fExpected.Close()
		bufExpected, errReadLocalExpected := ioutil.ReadAll(fExpected)
		if errReadLocalExpected != nil {
			t.Error("Error reading the local expected result file. Err: ", errReadLocalExpected)
		}

		if !testBinary(buf, bufExpected) {
			t.Errorf("Different data from original and uploaded file, got: %d (bytes), want: %d (bytes).", len(buf), len(bufExpected))
		}

		// Send response to be tested
		rw.Write([]byte(`OK`))
	}
	// Close the server when test finishes
	server := httptest.NewServer(http.HandlerFunc(serverHandleTest))
	defer server.Close()

	// Use test server data
	u, errURL := url.Parse(server.URL)
	if errURL != nil {
		t.Error("Error parsing test server URL. Err ", errURL)
	}
	up := New(nil, false, u.Scheme, u.Host, 3, 100)

	// Upload test file
	h := map[string]string{headerName: headerValue}
	errUpload := up.UploadLocalFile(testFilePath, UploadFilePath, h)
	if errUpload != nil {
		t.Error("Error uploading localfile. Err ", errUpload)
	}
}

func TestUploadData(t *testing.T) {
	data := []byte("ABCDE")
	headerName := "hName"
	headerValue := "hValue"
	UploadFilePath := "test/fileData.ts"

	serverHandleTest := func(rw http.ResponseWriter, req *http.Request) {
		// Check path
		if req.URL.Path != "/"+UploadFilePath {
			t.Errorf("Server received wrong path, got: %s, want: %s.", req.URL.String(), UploadFilePath)
		}

		// Check headers
		hVal := req.Header.Get(headerName)
		if hVal == "" {
			t.Errorf("Header not received, got: \"\", want: %s.", headerName)
		} else if hVal != headerValue {
			t.Errorf("Header value received is wrong, got: %s, want: %s.", hVal, headerValue)
		}

		// Check data
		buf, errReadReq := ioutil.ReadAll(req.Body)
		if errReadReq != nil {
			t.Error("Error reading the sent body. Err: ", errReadReq)
		}

		if !testBinary(buf, data) {
			t.Errorf("Different data from original and uploaded file, got: %d (bytes), want: %d (bytes).", len(buf), len(data))
		}

		// Send response to be tested
		rw.Write([]byte(`OK`))
	}
	// Close the server when test finishes
	server := httptest.NewServer(http.HandlerFunc(serverHandleTest))
	defer server.Close()

	// Use test server data
	u, errURL := url.Parse(server.URL)
	if errURL != nil {
		t.Error("Error parsing test server URL. Err ", errURL)
	}
	up := New(nil, false, u.Scheme, u.Host, 3, 100)

	// Upload test file
	h := map[string]string{headerName: headerValue}
	errUpload := up.UploadData(data, UploadFilePath, h)
	if errUpload != nil {
		t.Error("Error uploading data. Err ", errUpload)
	}
}

func TestUploadChunkedData(t *testing.T) {
	dataChunk1 := []byte("ABCDE")
	dataChunk2 := []byte("123456")
	headerName := "hName"
	headerValue := "hValue"
	UploadFilePath := "test/fileChunked.ts"

	// Need to wait until the req us processed
	var wg sync.WaitGroup
	wg.Add(1)

	serverHandleTest := func(rw http.ResponseWriter, req *http.Request) {
		// Check path
		if req.URL.Path != "/"+UploadFilePath {
			t.Errorf("Server received wrong path, got: %s, want: %s.", req.URL.String(), UploadFilePath)
		}

		// Check headers
		hVal := req.Header.Get(headerName)
		if hVal == "" {
			t.Errorf("Header not received, got: \"\", want: %s.", headerName)
		} else if hVal != headerValue {
			t.Errorf("Header value received is wrong, got: %s, want: %s.", hVal, headerValue)
		}

		// Check data
		buf := []byte{}
		bufChunk := make([]byte, 3)
		for {
			nRead, errRead := req.Body.Read(bufChunk)
			if errRead == io.EOF {
				break
			} else if errRead != nil {
				t.Error("Error reading the sent chunks. Err: ", errRead)
				break
			} else {
				buf = append(buf, bufChunk[0:nRead]...)
			}
		}
		totalData := append(dataChunk1, dataChunk2...)
		if !testBinary(buf, totalData) {
			t.Errorf("Different data from original and uploaded file, got: %d (bytes), want: %d (bytes).", len(buf), len(totalData))
		}

		// Send response to be tested
		rw.Write([]byte(`OK`))

		// Signal the final waiting
		wg.Done()
	}
	// Close the server when test finishes
	server := httptest.NewServer(http.HandlerFunc(serverHandleTest))
	defer server.Close()

	// Use test server data
	u, errURL := url.Parse(server.URL)
	if errURL != nil {
		t.Error("Error parsing test server URL. Err ", errURL)
	}
	up := New(nil, false, u.Scheme, u.Host, 3, 100)

	// Create channel
	h := map[string]string{headerName: headerValue}
	channel := up.UploadChunkedTransfer(UploadFilePath, h)
	if channel == nil {
		t.Error("Error creating channel")
	}

	// Upload 2 chunks
	channel <- dataChunk1
	channel <- dataChunk2
	close(channel)

	// Wait to process the data
	wg.Wait()
}
