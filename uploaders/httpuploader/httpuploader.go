package httpuploader

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// HTTPUploader HTTP uploader class class
type HTTPUploader struct {
	HTTPClient *http.Client

	Log                     *logrus.Logger
	HTTPSInsecure           bool
	HTTPScheme              string
	HTTPHost                string
	MaxHTTPRetries          int
	InitialHTTPRetryDelayMs int
}

// New Creates a chunk instance
func New(log *logrus.Logger, httpsInsecure bool, httpScheme string, httpHost string, maxHTTPRetries int, initialHTTPRetryDelayMs int) HTTPUploader {
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.DebugLevel)
	}

	var tr = http.DefaultTransport
	if (strings.Compare(httpScheme, "https") == 0) && (httpsInsecure) {
		// Setup HTTPS client in dev env, skips CA verification
		log.Warn("Skipping CA cert verification!")
		tlsConfig := &tls.Config{InsecureSkipVerify: true}
		tr = &http.Transport{TLSClientConfig: tlsConfig}
	}
	client := http.Client{
		Transport: tr,
		Timeout:   0,
	}
	h := HTTPUploader{&client, log, httpsInsecure, httpScheme, httpHost, maxHTTPRetries, initialHTTPRetryDelayMs}

	return h
}

// UploadLocalFile Uploads a file from the filesystem
func (h *HTTPUploader) UploadLocalFile(localFilename string, dstPathFile string, headers map[string]string) error {
	f, errOpen := os.Open(localFilename)
	defer f.Close()
	if errOpen != nil {
		h.Log.Error("ERROR reading  ", localFilename, "(", dstPathFile, ")")
		return errOpen
	}

	return h.uploadDataRetries(f, dstPathFile, headers)
}

// UploadData Uploads data array
func (h *HTTPUploader) UploadData(data []byte, dstPathFile string, headers map[string]string) error {
	return h.uploadDataRetries(bytes.NewReader(data), dstPathFile, headers)
}

// UploadChunkedTransfer Uploads data as soon as arrives to the returned channel (no retries for chunked transfer, future improvement)
func (h *HTTPUploader) UploadChunkedTransfer(dstPathFile string, headers map[string]string) chan []byte {
	r, w := io.Pipe()
	writeChan := make(chan []byte)

	// open request
	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: h.HTTPScheme,
			Host:   h.HTTPHost,
			Path:   "/" + dstPathFile,
		},
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: -1,
		Body:          r,
		Header:        http.Header{},
	}

	// Add headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	go func() {
		defer w.Close()

		for buf := range writeChan {
			n, err := w.Write(buf)
			h.Log.Debug("Wrote ", n, " bytes to ", dstPathFile)
			if n != len(buf) && err != nil {
				panic(err)
			}
		}
	}()

	go func() {
		h.Log.Debug("Opening connection to upload to ", dstPathFile)
		h.Log.Debug("Req: ", req)
		_, err := h.HTTPClient.Do(req)

		if err != nil {
			h.Log.Error("Error uploading to ", dstPathFile, ". Error: ", err)
		} else {
			h.Log.Debug("Upload to ", dstPathFile, " complete")
		}
	}()

	return writeChan
}

func (h *HTTPUploader) uploadDataRetries(dataReader io.Reader, dstPathFile string, headers map[string]string) error {
	var ret error = nil
	maxRetries := h.MaxHTTPRetries
	retryPauseInitialMs := h.InitialHTTPRetryDelayMs
	retryIntent := 0

	for {
		if retryIntent >= maxRetries {
			h.Log.Error("ERROR data lost because server busy, ", dstPathFile)
			break
		} else {
			retryErr := h.uploadData(dataReader, dstPathFile, headers)
			if retryErr != nil {
				time.Sleep(time.Duration(retryPauseInitialMs*retryIntent) * time.Millisecond)
			} else {
				break
			}
		}
		retryIntent++
	}

	return ret
}

func (h *HTTPUploader) uploadData(fileData io.Reader, dstPathFile string, headers map[string]string) error {
	var ret error = nil

	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: h.HTTPScheme,
			Host:   h.HTTPHost,
			Path:   "/" + dstPathFile,
		},
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: -1,
		Body:          ioutil.NopCloser(fileData),
		Header:        http.Header{},
	}

	// Add headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, errReq := h.HTTPClient.Do(req)
	if errReq != nil {
		h.Log.Error("Error uploading to ", dstPathFile, ")", "Error: ", errReq)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode < 400 {
			// Done
			h.Log.Info("Upload to ", dstPathFile, " complete")
		} else if resp.StatusCode == http.StatusServiceUnavailable {
			// Need to retry
			h.Log.Debug("Warning server busy, uploading to ", dstPathFile, ", RETRYING!")
			ret = errors.New("Retryable upload error")
		} else {
			// Not retirable error
			h.Log.Error("Error server uploading to ", dstPathFile, ")", "HTTP Error: ", resp.StatusCode)
		}
	}

	return ret
}
