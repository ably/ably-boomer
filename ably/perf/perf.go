package perf

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const _defaultKeyPrefix = "perf"

// S3ObjectPutter provides a PutObject function for writing to S3
type S3ObjectPutter interface {
	PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

// Reporter provides profiling and performance debugging instrumentation
type Reporter struct {
	started       bool
	config        *Config
	s3Client      S3ObjectPutter
	pprofFile     *os.File
	pprofFileName string
	hist          *Histogram
	histFileName  string
}

// Only allow alphanumeric chars, - _ and . in file names
var replaceChars = regexp.MustCompile("[^a-zA-Z0-9-_.]")

// NewReporter creates a new instance of a Reporter with defaults
func NewReporter() *Reporter {
	return &Reporter{}
}

// NewReporterWithS3 creates a new instance of a Perf with defaults and a
// supplied S3 client override
func NewReporterWithS3(config *Config, s3Client S3ObjectPutter) *Reporter {
	return &Reporter{
		config:   config,
		s3Client: s3Client,
	}
}

// Start begins profiling based on the environment configuration. Start should
// be called at most once. If start is called, stop must be called.
func (p *Reporter) Start() error {
	if p.started {
		return fmt.Errorf("perf is already started")
	}

	if p.config == nil {
		perfConfig, perfConfigErr := NewConfig(os.LookupEnv)

		if perfConfigErr != nil {
			return perfConfigErr
		}

		p.config = perfConfig
	}

	hostname, hostnameErr := os.Hostname()
	if hostnameErr != nil {
		hostname = "unknown"
	}
	now := time.Now().Unix()

	if p.config.HistogramDir != "" {
		p.hist = NewDefaultHistogram()
		histBaseName := replaceChars.ReplaceAllString(
			fmt.Sprintf(
				"latency-%s-%d.hist",
				hostname,
				now,
			),
			"_",
		)
		p.histFileName = path.Join(p.config.HistogramDir, histBaseName)
	}

	if p.config.CPUProfileDir == "" {
		return nil
	}

	pprofBaseName := replaceChars.ReplaceAllString(
		fmt.Sprintf(
			"cpuprofile-%s-%d.pprof",
			hostname,
			now,
		),
		"_",
	)
	p.pprofFileName = path.Join(p.config.CPUProfileDir, pprofBaseName)
	f, fErr := os.Create(p.pprofFileName)
	if fErr != nil {
		return fErr
	}

	p.pprofFile = f
	p.started = true
	pprof.StartCPUProfile(f)

	return nil
}

// RecordSuccess reports a success.
func (p *Reporter) RecordSuccess(
	requestType string,
	name string,
	responseTime int64,
	responseLength int64,
) {
	boomer.RecordSuccess(requestType, name, responseTime, responseLength)
	// TODO: record extra details to logs/histogram
}

// RecordFailure reports a failure
func (p *Reporter) RecordFailure(
	requestType string,
	name string,
	responseTime int64,
	exception string,
) {
	boomer.RecordFailure(requestType, name, responseTime, exception)
	// TODO: record extra details to logs/histogram
}

// Stop will stop any profiling that was started and write the files to the
// configured locations (disk and s3). Stop may be called multiple times so
// it is safe to both call stop directly and defer calls to stop.
func (p *Reporter) Stop() error {
	if !p.started {
		return nil
	}
	p.started = false

	errors := []string(nil)

	pprofErr := p.stopPProf()
	if pprofErr != nil {
		errors = append(errors, pprofErr.Error())
	}
	histErr := p.stopHist()
	if histErr != nil {
		errors = append(errors, histErr.Error())
	}

	if errors != nil {
		return fmt.Errorf(strings.Join(errors, ", "))
	}

	return nil
}

func (p *Reporter) stopPProf() error {
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

	if p.config.CPUProfileS3Bucket != "" {
		s3Err := p.uploadToS3(p.pprofFile.Name(), p.config.CPUProfileS3Bucket)
		if s3Err != nil {
			return fmt.Errorf("error uploading pprof file to s3: %s", s3Err)
		}
	}

	return nil
}

func (p *Reporter) stopHist() error {
	if p.hist == nil {
		return nil
	}

	histFile, histFileErr := os.Create(p.histFileName)
	if histFileErr != nil {
		return fmt.Errorf("error opening histogram file: %s", histFileErr)
	}
	defer histFile.Close()

	histWriter := NewHistogramWriter(histFile)
	writeErr := histWriter.Write(p.hist)
	if writeErr != nil {
		return fmt.Errorf("error writing histogram to file: %s", writeErr)
	}

	syncErr := histFile.Sync()
	if syncErr != nil {
		return fmt.Errorf("error syncing histogram file: %s", syncErr)
	}

	closeErr := histFile.Close()
	if closeErr != nil {
		return fmt.Errorf("error closing histogram file: %s", closeErr)
	}

	if p.config.HistogramS3Bucket != "" {
		s3Err := p.uploadToS3(histFile.Name(), p.config.HistogramS3Bucket)
		if s3Err != nil {
			return fmt.Errorf("error uploading histogram file to s3: %s", s3Err)
		}
	}

	return nil
}

// Returns either the configured s3 client or the default s3 client if unset
func (p *Reporter) configuredS3Client() (S3ObjectPutter, error) {
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
func (p *Reporter) uploadToS3(fileName string, bucket string) error {
	s3Client, s3ClientErr := p.configuredS3Client()
	if s3ClientErr != nil {
		return s3ClientErr
	}

	file, fileErr := os.Open(fileName)
	if fileErr != nil {
		return fileErr
	}
	defer file.Close()

	key := path.Join(_defaultKeyPrefix, path.Base(fileName))

	// Get file size and read the file content into a buffer
	fileInfo, fileInfoErr := file.Stat()
	if fileInfoErr != nil {
		return fmt.Errorf("error reading file stat: %s", fileInfoErr)
	}
	size := fileInfo.Size()

	_, s3Err := s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(bucket),
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
