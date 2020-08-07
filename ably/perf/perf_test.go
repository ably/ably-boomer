package perf

import (
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
)

func TestPerfPProf(t *testing.T) {
	t.Run("pprof works with sensible defaults", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_CPU_PROFILE_DIR":       os.TempDir(),
			"PERF_CPU_PROFILE_S3_BUCKET": "ably-logs-dev",
		}

		config, configErr := NewConfig(testEnv.LookupEnv)
		if configErr != nil {
			ts.Fatalf("failed to initialize perf config: %s", configErr)
		}

		reporter := &mockBoomer{}
		s3Client := &mockS3{}

		perf := NewReporter(config, reporter, s3Client)
		perf.Start()
		defer cleanup(ts, perf)

		time.Sleep(100 * time.Millisecond)

		perfErr := perf.Stop()
		if perfErr != nil {
			ts.Fatalf("error stopping perf: %s", perfErr)
		}

		// Test that a cpuprofile was written to disk
		fileExt := path.Ext(perf.pprofFileName)
		expectedFileExt := ".pprof"
		if fileExt != expectedFileExt {
			ts.Fatalf(
				"unexpected pprof extension, got: %s, wanted: %s",
				fileExt,
				expectedFileExt,
			)
		}
		pprofStat, pprofStatErr := os.Stat(perf.pprofFileName)
		if pprofStatErr != nil {
			ts.Fatalf(
				"pprof file missing from disk: %s",
				pprofStatErr,
			)
		} else if pprofStat.Size() == 0 {
			ts.Fatalf("pprof file is empty")
		}

		// Test that the pprof file was uploaded to the ably-logs-dev s3 bucket
		if s3Client.input == nil {
			ts.Fatalf("s3 PutObject was not called")
		}

		// S3 Bucket
		bucket := s3Client.input.Bucket
		expectedBucket := "ably-logs-dev"
		if bucket == nil {
			ts.Errorf("missing bucket name is s3 client options")
		} else if *bucket != expectedBucket {
			ts.Errorf(
				"unexpected s3 bucket, got: %s, wanted: %s",
				*bucket,
				expectedBucket,
			)
		}

		// S3 Key
		key := s3Client.input.Key
		expectedKey := path.Join(
			_defaultPprofKeyPrefix,
			path.Base(perf.pprofFileName),
		)
		if key == nil {
			ts.Errorf("missing key in s3 client options")
		} else if *key != expectedKey {
			ts.Errorf(
				"unexpected s3 key, got: %s, wanted: %s",
				*key,
				expectedKey,
			)
		}

		/* S3 ACL
		acl := s3Client.input.ACL
		expectedACL := "authenticated-read"
		if acl == nil {
			ts.Errorf("missing s3 file ACL")
		} else if *acl != expectedACL {
			ts.Errorf(
				"unexpected s3 file ACL, got: %s, wanted: %s",
				*acl,
				expectedACL,
			)
		} */

		// S3 Body
		s3File, s3FileOk := s3Client.input.Body.(*os.File)
		if !s3FileOk || s3File == nil {
			ts.Errorf("missing file as s3 PutObject body")
		} else if s3File.Name() != perf.pprofFileName {
			ts.Errorf(
				"unexpected s3 file upload, got: %s, wanted: %s",
				s3File.Name(),
				perf.pprofFileName,
			)
		}

		// S3 ContentLength
		s3ContentLength := s3Client.input.ContentLength
		expectedS3ContentLength := pprofStat.Size()
		if s3ContentLength == nil {
			ts.Errorf("missing s3 file content length")
		} else if *s3ContentLength != expectedS3ContentLength {
			ts.Errorf(
				"unexpected s3 content length: got %d, wanted %d",
				*s3ContentLength,
				expectedS3ContentLength,
			)
		}

		// S3 ContentType
		s3ContentType := s3Client.input.ContentType
		expectedS3ContentType := "application/octet-stream"
		if s3ContentType == nil {
			ts.Errorf("missing content type in s3 client options")
		} else if *s3ContentType != expectedS3ContentType {
			ts.Errorf(
				"unexpected s3 content type, got: %s, wanted: %s",
				*s3ContentType,
				expectedS3ContentType,
			)
		}
	})

	t.Run("pprof does not write to s3 unless configured", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_CPU_PROFILE_DIR": os.TempDir(),
		}

		config, configErr := NewConfig(testEnv.LookupEnv)
		if configErr != nil {
			ts.Fatalf("failed to initialize perf config: %s", configErr)
		}

		reporter := &mockBoomer{}

		s3Client := &mockS3{}

		perf := NewReporter(config, reporter, s3Client)
		perf.Start()
		defer cleanup(ts, perf)

		time.Sleep(100 * time.Millisecond)

		perfErr := perf.Stop()
		if perfErr != nil {
			ts.Fatalf("error stopping perf: %s", perfErr)
		}

		// Test that a cpuprofile was written to disk
		fileExt := path.Ext(perf.pprofFileName)
		expectedFileExt := ".pprof"
		if fileExt != expectedFileExt {
			ts.Fatalf(
				"unexpected pprof extension, got: %s, wanted: %s",
				fileExt,
				expectedFileExt,
			)
		}
		pprofStat, pprofStatErr := os.Stat(perf.pprofFileName)
		if pprofStatErr != nil {
			ts.Fatalf(
				"pprof file missing from disk: %s",
				pprofStatErr,
			)
		} else if pprofStat.Size() == 0 {
			ts.Fatalf("pprof file is empty")
		}

		// Test that the pprof file was uploaded to the ably-logs-dev s3 bucket
		if s3Client.input != nil {
			ts.Fatalf("s3 PutObject should not be called")
		}
	})

	t.Run("pprof doesn't run by default", func(ts *testing.T) {
		// Check that the environment doesn't contain perf configuration
		profileDir, profileDirSet := os.LookupEnv("PERF_CPU_PROFILE_DIR")
		if profileDirSet && profileDir != "" {
			ts.Fatalf(
				"PERF_CPU_PROFILE_DIR env is currently set: %s",
				profileDir,
			)
		}

		bucket, bucketSet := os.LookupEnv("PERF_CPU_PROFILE_S3_BUCKET")
		if bucketSet && bucket != "" {
			ts.Fatalf(
				"PERF_CPU_PROFILE_S3_BUCKET env is currently set: %s",
				bucket,
			)
		}

		perf := NewDefaultReporter()
		perf.Start()
		defer cleanup(ts, perf)

		time.Sleep(100 * time.Millisecond)

		perfErr := perf.Stop()
		if perfErr != nil {
			ts.Fatalf("error stopping perf: %s", perfErr)
		}

		// Test that a cpuprofile is not taken by default
		if perf.pprofFileName != "" {
			ts.Fatalf("expected no pprof file by default")
		}
	})

	t.Run("pprof defaults to real s3 client", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_CPU_PROFILE_DIR":       os.TempDir(),
			"PERF_CPU_PROFILE_S3_BUCKET": "ably-logs-dev",
		}

		config, configErr := NewConfig(testEnv.LookupEnv)
		if configErr != nil {
			ts.Fatalf("failed to initialize perf config: %s", configErr)
		}

		perf := NewReporter(config, nil, nil)

		configuredS3Client, configuredS3ClientErr := perf.configuredS3Client()
		if configuredS3ClientErr != nil {
			ts.Fatalf(
				"unexpected error getting s3 client: %s",
				configuredS3ClientErr,
			)
		} else {
			s3Client, s3ClientOK := configuredS3Client.(*s3.S3)
			if !s3ClientOK || s3Client == nil {
				ts.Fatalf("expected s3 client to default to a real client")
			}
		}
	})
}

func TestPerfHistogram(t *testing.T) {
	t.Run("histograms work with sensible defaults", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_HISTOGRAM_DIR":       os.TempDir(),
			"PERF_HISTOGRAM_S3_BUCKET": "ably-logs-dev",
		}

		config, configErr := NewConfig(testEnv.LookupEnv)
		if configErr != nil {
			ts.Fatalf("failed to initialize perf config: %s", configErr)
		}

		reporter := &mockBoomer{}
		s3Client := &mockS3{}

		perf := NewReporter(config, reporter, s3Client)
		perf.Start()
		defer cleanup(ts, perf)

		expectedHistMap := writeTestRecords(perf, 100)

		perfErr := perf.Stop()
		if perfErr != nil {
			ts.Fatalf("error stopping perf: %s", perfErr)
		}

		// Test that the histograms are as expected
		assertEqualHistogramFile(ts, perf.histFileName, expectedHistMap)

		// Test that the pprof file was uploaded to the ably-logs-dev s3 bucket
		histStat, histStatErr := os.Stat(perf.histFileName)
		if histStatErr != nil {
			t.Fatalf(
				"histogram file missing from disk: %s",
				histStatErr,
			)
		} else if histStat.Size() == 0 {
			t.Fatalf("histogram file is empty")
		}

		if s3Client.input == nil {
			ts.Fatalf("s3 PutObject was not called")
		}

		// S3 Bucket
		bucket := s3Client.input.Bucket
		expectedBucket := "ably-logs-dev"
		if bucket == nil {
			ts.Errorf("missing bucket name is s3 client options")
		} else if *bucket != expectedBucket {
			ts.Errorf(
				"unexpected s3 bucket, got: %s, wanted: %s",
				*bucket,
				expectedBucket,
			)
		}

		// S3 Key
		key := s3Client.input.Key
		expectedKey := path.Join(
			_defaultHistKeyPrefix,
			path.Base(perf.histFileName),
		)
		if key == nil {
			ts.Errorf("missing key in s3 client options")
		} else if *key != expectedKey {
			ts.Errorf(
				"unexpected s3 key, got: %s, wanted: %s",
				*key,
				expectedKey,
			)
		}

		/* S3 ACL
		acl := s3Client.input.ACL
		expectedACL := "authenticated-read"
		if acl == nil {
			ts.Errorf("missing s3 file ACL")
		} else if *acl != expectedACL {
			ts.Errorf(
				"unexpected s3 file ACL, got: %s, wanted: %s",
				*acl,
				expectedACL,
			)
		} */

		// S3 Body
		s3File, s3FileOk := s3Client.input.Body.(*os.File)
		if !s3FileOk || s3File == nil {
			ts.Errorf("missing file as s3 PutObject body")
		} else if s3File.Name() != perf.histFileName {
			ts.Errorf(
				"unexpected s3 file upload, got: %s, wanted: %s",
				s3File.Name(),
				perf.pprofFileName,
			)
		}

		// S3 ContentLength
		s3ContentLength := s3Client.input.ContentLength
		expectedS3ContentLength := histStat.Size()
		if s3ContentLength == nil {
			ts.Errorf("missing s3 file content length")
		} else if *s3ContentLength != expectedS3ContentLength {
			ts.Errorf(
				"unexpected s3 content length: got %d, wanted %d",
				*s3ContentLength,
				expectedS3ContentLength,
			)
		}

		// S3 ContentType
		s3ContentType := s3Client.input.ContentType
		expectedS3ContentType := "application/octet-stream"
		if s3ContentType == nil {
			ts.Errorf("missing content type in s3 client options")
		} else if *s3ContentType != expectedS3ContentType {
			ts.Errorf(
				"unexpected s3 content type, got: %s, wanted: %s",
				*s3ContentType,
				expectedS3ContentType,
			)
		}
	})

	t.Run("hist does not write to s3 unless configured", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_HISTOGRAM_DIR": os.TempDir(),
		}

		config, configErr := NewConfig(testEnv.LookupEnv)
		if configErr != nil {
			ts.Fatalf("failed to initialize perf config: %s", configErr)
		}

		reporter := &mockBoomer{}

		s3Client := &mockS3{}

		perf := NewReporter(config, reporter, s3Client)
		perf.Start()
		defer cleanup(ts, perf)

		expectedHistMap := writeTestRecords(perf, 100)

		perfErr := perf.Stop()
		if perfErr != nil {
			ts.Fatalf("error stopping perf: %s", perfErr)
		}

		assertEqualHistogramFile(ts, perf.histFileName, expectedHistMap)

		// Test that the pprof file was uploaded to the ably-logs-dev s3 bucket
		if s3Client.input != nil {
			ts.Fatalf("s3 PutObject should not be called")
		}
	})

	t.Run("histogram doesn't run by default", func(ts *testing.T) {
		// Check that the environment doesn't contain perf configuration
		profileDir, profileDirSet := os.LookupEnv("PERF_HISTOGRAM_DIR")
		if profileDirSet && profileDir != "" {
			ts.Fatalf(
				"PERF_HISTOGRAM_DIR env is currently set: %s",
				profileDir,
			)
		}

		bucket, bucketSet := os.LookupEnv("PERF_HISTOGRAM_S3_BUCKET")
		if bucketSet && bucket != "" {
			ts.Fatalf(
				"PERF_HISTOGRAM_S3_BUCKET env is currently set: %s",
				bucket,
			)
		}

		perf := NewDefaultReporter()
		perf.Start()
		defer cleanup(ts, perf)

		time.Sleep(100 * time.Millisecond)

		perfErr := perf.Stop()
		if perfErr != nil {
			ts.Fatalf("error stopping perf: %s", perfErr)
		}

		// Test that a cpuprofile is not taken by default
		if perf.pprofFileName != "" {
			ts.Fatalf("expected no pprof file by default")
		}
	})

	t.Run("histogram defaults to real s3 client", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_HISTOGRAM_DIR":    os.TempDir(),
			"PERF_HISTOGRAM_BUCKET": "ably-logs-dev",
		}

		config, configErr := NewConfig(testEnv.LookupEnv)
		if configErr != nil {
			ts.Fatalf("failed to initialize perf config: %s", configErr)
		}

		perf := NewReporter(config, nil, nil)

		configuredS3Client, configuredS3ClientErr := perf.configuredS3Client()
		if configuredS3ClientErr != nil {
			ts.Fatalf(
				"unexpected error getting s3 client: %s",
				configuredS3ClientErr,
			)
		} else {
			s3Client, s3ClientOK := configuredS3Client.(*s3.S3)
			if !s3ClientOK || s3Client == nil {
				ts.Fatalf("expected s3 client to default to a real client")
			}
		}
	})
}

func writeTestRecords(r LocustReporter, count int) map[string]*Histogram {
	expectedHistMap := map[string]*Histogram{}

	for i := 0; i < 100; i++ {
		roundTripTime := int64(rand.Intn(60001))
		failureTime := int64(rand.Intn(60001))
		r.RecordSuccess("ably", "subscribe", roundTripTime, 100)
		r.RecordFailure("ably", "subscribe", failureTime, "testing")

		sHist, ok := expectedHistMap["ably.subscribe.success"]
		if !ok {
			sHist = NewDefaultHistogram()
			expectedHistMap["ably.subscribe.success"] = sHist
		}
		sHist.Add(roundTripTime)

		fHist, ok := expectedHistMap["ably.subscribe.failure"]
		if !ok {
			fHist = NewDefaultHistogram()
			expectedHistMap["ably.subscribe.failure"] = fHist
		}
		fHist.Add(failureTime)
	}

	return expectedHistMap
}

// Mock boomer records the success and failure calls by default
type mockBoomer struct {
	success []*mockBoomerSuccess
	failure []*mockBoomerFailure
}

func (b *mockBoomer) RecordSuccess(
	requestType string,
	name string,
	responseTime int64,
	responseLength int64,
) {
	b.success = append(b.success, newMockBoomerSuccess(
		requestType,
		name,
		responseTime,
		responseLength,
	))
}
func (b *mockBoomer) RecordFailure(
	requestType string,
	name string,
	responseTime int64,
	exception string,
) {
	b.failure = append(b.failure, newMockBoomerFailure(
		requestType,
		name,
		responseTime,
		exception,
	))
}

type mockBoomerSuccess struct {
	requestType    string
	name           string
	responseTime   int64
	responseLength int64
}

func newMockBoomerSuccess(
	requestType string,
	name string,
	responseTime int64,
	responseLength int64,
) *mockBoomerSuccess {
	return &mockBoomerSuccess{
		requestType,
		name,
		responseTime,
		responseLength,
	}
}

type mockBoomerFailure struct {
	requestType  string
	name         string
	responseTime int64
	exception    string
}

func newMockBoomerFailure(
	requestType string,
	name string,
	responseTime int64,
	exception string,
) *mockBoomerFailure {
	return &mockBoomerFailure{
		requestType,
		name,
		responseTime,
		exception,
	}
}

type mockS3 struct {
	err    error
	input  *s3.PutObjectInput
	output *s3.PutObjectOutput
}

func (s *mockS3) PutObject(
	input *s3.PutObjectInput,
) (*s3.PutObjectOutput, error) {
	s.input = input
	if s.err != nil {
		return nil, s.err
	}

	return s.output, nil
}

func cleanup(t *testing.T, perf *Reporter) {
	err := perf.Stop()
	// Cleanup the pprof file if we can
	if path.Ext(perf.pprofFileName) == ".pprof" {
		os.Remove(perf.pprofFileName)
	}
	if path.Ext(perf.histFileName) == ".hist" {
		os.Remove(perf.histFileName)
	}
	if err != nil {
		t.Fatalf("error stopping perf: %s", err)
	}
}
