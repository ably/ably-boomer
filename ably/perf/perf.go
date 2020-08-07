package perf

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/ably-forks/boomer"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const _defaultPprofKeyPrefix = "perf"
const _defaultHistKeyPrefix = "hist"

// Verif that Perf satisfies the LocustReporter interface
var _ LocustReporter = LocustReporter(&Reporter{})

// S3ObjectPutter provides a PutObject function for writing to S3
type S3ObjectPutter interface {
	PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
}

// HistMap key indexes histograms by request type and name
type histMapKey struct {
	requestType, name, report string
}

// A latency record contains a histogram entry
type latencyRecord struct {
	key          histMapKey
	responseTime int64
}

func (h *histMapKey) ID() string {
	return strings.Join([]string{h.requestType, h.name, h.report}, ".")
}

// Reporter provides profiling and performance debugging instrumentation
type Reporter struct {
	startTime     int64
	pprofStarted  bool
	histStarted   bool
	config        *Config
	boomer        LocustReporter
	s3Client      S3ObjectPutter
	pprofFile     *os.File
	pprofFileName string
	hist          map[histMapKey]*Histogram
	histChan      chan *latencyRecord
	histChanLock  sync.RWMutex
	histWG        sync.WaitGroup
	histMapKey    histMapKey
	histFileName  string
}

// Only allow alphanumeric chars, - _ and . in file names
var replaceChars = regexp.MustCompile("[^a-zA-Z0-9-_.]")

// NewDefaultReporter creates a new instance of a Reporter with defaults
func NewDefaultReporter() *Reporter {
	return &Reporter{}
}

// NewReporter creates a new instance of a Perf with the supplied config and
// clients
func NewReporter(
	config *Config,
	boomer LocustReporter,
	s3Client S3ObjectPutter,
) *Reporter {
	return &Reporter{
		config:   config,
		boomer:   boomer,
		s3Client: s3Client,
	}
}

// Start begins profiling based on the environment configuration. Start should
// be called at most once. If start is called, stop must be called.
func (p *Reporter) Start() error {
	errors := []string(nil)
	p.startTime = time.Now().Unix()

	pprofErr := p.startPProf()
	if pprofErr != nil {
		errors = append(errors, pprofErr.Error())
	}
	histErr := p.startHist()
	if histErr != nil {
		errors = append(errors, histErr.Error())
	}

	if errors != nil {
		return fmt.Errorf(strings.Join(errors, ", "))
	}

	return nil
}

func (p *Reporter) startPProf() error {
	if p.pprofStarted {
		return fmt.Errorf("pprof is already started")
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
	now := p.startTime

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
	p.pprofStarted = true
	pprof.StartCPUProfile(f)

	return nil
}

func (p *Reporter) startHist() error {
	if p.histStarted {
		return fmt.Errorf("histogram is already started")
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
	now := p.startTime

	if p.config.HistogramDir != "" {
		p.hist = map[histMapKey]*Histogram{}
		histBaseName := replaceChars.ReplaceAllString(
			fmt.Sprintf(
				"latency-%s-%d.hist",
				hostname,
				now,
			),
			"_",
		)
		p.histFileName = path.Join(p.config.HistogramDir, histBaseName)
		p.histChan = make(chan *latencyRecord)
		p.histWG.Add(1)
		go p.recordLatencies()
		p.histStarted = true
	}

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
	p.histChanLock.RLock()
	defer p.histChanLock.RUnlock()
	if p.histStarted {
		p.histChan <- &latencyRecord{
			key:          histMapKey{requestType, name, "success"},
			responseTime: responseTime,
		}
	}
}

// RecordFailure reports a failure
func (p *Reporter) RecordFailure(
	requestType string,
	name string,
	responseTime int64,
	exception string,
) {
	boomer.RecordFailure(requestType, name, responseTime, exception)
	p.histChanLock.RLock()
	defer p.histChanLock.RUnlock()
	if p.histStarted {
		p.histChan <- &latencyRecord{
			key:          histMapKey{requestType, name, "failure"},
			responseTime: responseTime,
		}
	}
}

// Stop will stop any profiling that was started and write the files to the
// configured locations (disk and s3). Stop may be called multiple times so
// it is safe to both call stop directly and defer calls to stop.
func (p *Reporter) Stop() error {
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
	if !p.pprofStarted {
		return nil
	}
	defer p.pprofFile.Close()
	p.pprofStarted = false

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
		s3Err := p.uploadToS3(
			_defaultPprofKeyPrefix,
			p.pprofFile.Name(),
			p.config.CPUProfileS3Bucket,
		)
		if s3Err != nil {
			return fmt.Errorf("error uploading pprof file to s3: %s", s3Err)
		}
	}

	return nil
}

func (p *Reporter) stopHist() error {
	if !p.histStarted {
		return nil
	}

	p.histChanLock.Lock()
	p.histStarted = false
	close(p.histChan)
	p.histChanLock.Unlock()
	p.histWG.Wait()

	histFile, histFileErr := os.Create(p.histFileName)
	if histFileErr != nil {
		return fmt.Errorf("error opening histogram file: %s", histFileErr)
	}
	defer histFile.Close()

	// Write all of the histograms to the gob stream
	histWriter := NewHistogramWriter(histFile)
	for key, hist := range p.hist {
		writeErr := histWriter.Write(key.ID(), hist)
		if writeErr != nil {
			return fmt.Errorf("error writing histogram to file: %s", writeErr)
		}
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
		s3Err := p.uploadToS3(
			_defaultHistKeyPrefix,
			histFile.Name(),
			p.config.HistogramS3Bucket,
		)
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
	p.s3Client = s3.New(sess)

	return p.s3Client, nil
}

// Uploads a file to the S3 perf bucket
func (p *Reporter) uploadToS3(
	prefix string,
	fileName string,
	bucket string,
) error {
	s3Client, s3ClientErr := p.configuredS3Client()
	if s3ClientErr != nil {
		return s3ClientErr
	}

	file, fileErr := os.Open(fileName)
	if fileErr != nil {
		return fileErr
	}
	defer file.Close()

	var key string
	if prefix == "" {
		key = path.Base(fileName)
	} else {
		safePrefix := replaceChars.ReplaceAllString(prefix, "_")
		key = path.Join(
			safePrefix,
			path.Base(fileName),
		)
	}

	// Get file size and read the file content into a buffer
	fileInfo, fileInfoErr := file.Stat()
	if fileInfoErr != nil {
		return fmt.Errorf("error reading file stat: %s", fileInfoErr)
	}
	size := fileInfo.Size()

	_, s3Err := s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(key),
		ACL:           aws.String("authenticated-read"),
		Body:          file,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String("application/octet-stream"),
	})

	if s3Err != nil {
		return fmt.Errorf("s3 PutObject returned error: %s", s3Err)
	}

	return nil
}

// recordLatencies inserts latency records into the correct histogram. This in
// executed inside a single goroutine to prevent race conditions on the
// histogram structures.
func (p *Reporter) recordLatencies() {
	for record := range p.histChan {
		histogram, ok := p.hist[record.key]

		if !ok {
			histogram = NewDefaultHistogram()
			p.hist[record.key] = histogram
		}

		histogram.Add(record.responseTime)
	}

	p.histWG.Done()
}
