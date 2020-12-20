package s3uploader

import (
	"bytes"
	"context"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
)

// S3Uploader HTTP uploader class class
type S3Uploader struct {
	S3Session *s3.S3

	Log                        *logrus.Logger
	S3Bucket                   string
	S3Region                   string
	S3UploadTimeOutMs          int
	S3GrantReadToUploadedFiles bool
	AWSCreds                   AWSLocalCreds
}

// AWSLocalCreds local creds for debugging
type AWSLocalCreds struct {
	Valid     bool
	AWSId     string
	AWSSecret string
}

// New Creates a chunk instance
func New(log *logrus.Logger, s3Bucket string, s3Region string, s3UploadTimeOutMs int, s3GrantReadToUploadedFiles bool, awsCreds AWSLocalCreds) S3Uploader {
	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.DebugLevel)
	}

	// All clients require a Session. The Session provides the client with
	// shared configuration such as region, endpoint, and credentials. A
	// Session should be shared where possible to take advantage of
	// configuration and credential caching. See the session package for
	// more information.
	var s3Session *s3.S3 = nil
	if awsCreds.Valid {
		creds := credentials.NewStaticCredentials(awsCreds.AWSId, awsCreds.AWSSecret, "")
		_, err := creds.Get()
		if err != nil {
			log.Error("ERROR getting local credentials with ID ", awsCreds.AWSId)
		}
		awsConfig := aws.NewConfig().WithRegion(s3Region).WithCredentials(creds)
		awsSession := session.New()
		s3Session = s3.New(awsSession, awsConfig)
	} else {
		awsSession := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState:       session.SharedConfigEnable,
			AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
		}))
		var awsConfig *aws.Config = nil
		if s3Region != "" {
			awsConfig = aws.NewConfig().WithRegion(s3Region)
		}
		s3Session = s3.New(awsSession, awsConfig)
	}
	return S3Uploader{s3Session, log, s3Bucket, s3Region, s3UploadTimeOutMs, s3GrantReadToUploadedFiles, awsCreds}
}

// UploadLocalFile Uploads a file from the filesystem
func (s *S3Uploader) UploadLocalFile(localFilename string, dstPathFile string, headers map[string]string) error {
	f, errOpen := os.Open(localFilename)
	if errOpen != nil {
		s.Log.Error("ERROR reading  ", localFilename, "(", s.S3Bucket, "/", dstPathFile, ")")
		return errOpen
	}
	defer f.Close()

	fileInfo, _ := f.Stat()
	var size int64 = fileInfo.Size()
	buffer := make([]byte, size)
	f.Read(buffer)

	return s.UploadData(buffer, dstPathFile, headers)
}

// UploadData upload bytes
func (s *S3Uploader) UploadData(buffer []byte, dstPathFile string, headers map[string]string) error {
	var ret error = nil

	// Create a context with a timeout that will abort the upload if it takes
	// more than the passed in timeout.
	ctx := context.Background()
	var cancelFn func()
	if s.S3UploadTimeOutMs > 0 {
		ctx, cancelFn = context.WithTimeout(ctx, time.Duration(s.S3UploadTimeOutMs)*time.Millisecond)
	}
	// Ensure the context is canceled to prevent leaking.
	// See context package for more information, https://golang.org/pkg/context/
	defer cancelFn()

	s3Obj := s3.PutObjectInput{
		Bucket: aws.String(s.S3Bucket),
		Key:    aws.String(dstPathFile),
		Body:   bytes.NewReader(buffer),
	}

	// Add headers & contentType
	meta := map[string]*string{}
	for k, v := range headers {
		// Content type
		if strings.ToLower(k) == "content-type" {
			s3Obj.ContentType = aws.String(v)
		} else {
			meta[k] = aws.String(v)
		}
	}
	s3Obj.Metadata = meta

	if s.S3GrantReadToUploadedFiles {
		s3Obj.ACL = aws.String("public-read")
	}

	// Uploads the object to S3. The Context will interrupt the request if the
	// timeout expires.
	_, s3Err := s.S3Session.PutObjectWithContext(ctx, &s3Obj)
	if s3Err != nil {
		awsErr, ok := s3Err.(awserr.Error)
		if ok && awsErr.Code() == request.CanceledErrorCode {
			// If the SDK can determine the request or retry delay was canceled
			// by a context the CanceledErrorCode error code will be returned.
			s.Log.Error("Error timeout uploading to ", s.S3Bucket, "/", dstPathFile, ". Err: ", awsErr)
		} else {
			// Final error
			s.Log.Error("Error uploading to ", s.S3Bucket, "/", dstPathFile, ". Err: ", awsErr)
		}
		ret = awsErr
	}
	return ret
}
