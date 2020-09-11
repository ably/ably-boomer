package perf

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime/pprof"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const defaultKeyPrefix = "perf"

// S3ObjectPutter provides a PutObject function for writing to S3
type S3ObjectPutter interface {
	PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

type Conf struct {
	CPUProfileDir string
	S3Bucket      string
}

// Perf provides profiling and performance debugging instrumentation
type Perf struct {
	started   bool
	conf      Conf
	s3Client  S3ObjectPutter
	pprofFile *os.File
	fileName  string
}

// Only allow alphanumeric chars, - _ and . in file names
var replaceChars = regexp.MustCompile("[^a-zA-Z0-9-_.]")

// New creates a new instance of a Perf with defaults
func New(conf Conf) *Perf {
	return &Perf{
		conf: conf,
	}
}

// NewWithS3 creates a new instance of a Perf with defaults and a supplied S3
// client override
func NewWithS3(conf Conf, s3Client S3ObjectPutter) *Perf {
	return &Perf{
		conf:     conf,
		s3Client: s3Client,
	}
}

// Start begins profiling based on the environment configuration. Start should
// be called at most once. If start is called, stop must be called.
func (p *Perf) Start() error {
	if p.started {
		return fmt.Errorf("perf is already started")
	}

	if p.conf.CPUProfileDir == "" {
		return nil
	}

	hostname, hostnameErr := os.Hostname()
	if hostnameErr != nil {
		hostname = "unknown"
	}
	baseName := replaceChars.ReplaceAllString(
		fmt.Sprintf(
			"cpuprofile-%s-%d.pprof",
			hostname,
			time.Now().Unix(),
		),
		"_",
	)
	p.fileName = path.Join(p.conf.CPUProfileDir, baseName)
	f, fErr := os.Create(p.fileName)
	if fErr != nil {
		return fErr
	}

	p.pprofFile = f
	p.started = true
	pprof.StartCPUProfile(f)

	return nil
}

// Stop will stop any profiling that was started and write the files to the
// configured locations (disk and s3). Stop may be called multiple times so
// it is safe to both call stop directly and defer calls to stop.
func (p *Perf) Stop() error {
	if !p.started {
		return nil
	}
	p.started = false
	defer p.pprofFile.Close()

	pprof.StopCPUProfile()
	syncErr := p.pprofFile.Sync()
	if syncErr != nil {
		return fmt.Errorf("error syncing pprof file: %s", syncErr)
	}

	closeErr := p.pprofFile.Close()
	if closeErr != nil {
		return fmt.Errorf("error closing pprof file: %s", closeErr)
	}

	if p.conf.S3Bucket != "" {
		return p.uploadToS3(p.pprofFile.Name())
	}

	return nil
}

// Returns either the configured s3 client or the default s3 client if unset
func (p *Perf) configuredS3Client() (S3ObjectPutter, error) {
	if p.s3Client != nil {
		return p.s3Client, nil
	}

	sess, sessErr := session.NewSession()
	if sessErr != nil {
		return nil, sessErr
	}

	return s3.New(sess), nil
}

// Uploads a file to the S3 perf bucket
func (p *Perf) uploadToS3(fileName string) error {
	s3Client, s3ClientErr := p.configuredS3Client()
	if s3ClientErr != nil {
		return s3ClientErr
	}

	file, fileErr := os.Open(fileName)
	if fileErr != nil {
		return fileErr
	}
	defer file.Close()

	key := path.Join(defaultKeyPrefix, path.Base(fileName))

	// Get file size and read the file content into a buffer
	fileInfo, fileInfoErr := file.Stat()
	if fileInfoErr != nil {
		return fmt.Errorf("error reading file stat: %s", fileInfoErr)
	}
	size := fileInfo.Size()

	_, s3Err := s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(p.conf.S3Bucket),
		Key:           aws.String(key),
		ACL:           aws.String("private"),
		Body:          file,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String("application/octet-stream"),
	})

	if s3Err != nil {
		return fmt.Errorf("s3 PutObject returned error: %s", s3Err)
	}

	return nil
}
