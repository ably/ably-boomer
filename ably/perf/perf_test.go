package perf

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
)

func TestNewPerf(t *testing.T) {
	t.Run("perf works with sensible defaults", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_CPU_PROFILE_DIR": os.TempDir(),
			"PERF_CPU_S3_BUCKET":   "ably-logs-dev",
		}

		config, configErr := NewConfig(testEnv.LookupEnv)
		if configErr != nil {
			ts.Fatalf("failed to initialize perf config: %s", configErr)
		}

		s3Client := &mockS3{}

		perf := NewReporterWithS3(config, s3Client)
		perf.Start()
		defer func() {
			err := perf.Stop()
			// Cleanup the pprof file if we can
			if path.Ext(perf.pprofFileName) == ".pprof" {
				os.Remove(perf.pprofFileName)
			}
			if err != nil {
				ts.Fatalf("error stopping perf: %s", err)
			}
		}()

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
			_defaultKeyPrefix,
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

		// S3 ACL
		acl := s3Client.input.ACL
		expectedACL := "private"
		if acl == nil {
			ts.Errorf("missing s3 file ACL")
		} else if *acl != expectedACL {
			ts.Errorf(
				"unexpected s3 file ACL, got: %s, wanted: %s",
				*acl,
				expectedACL,
			)
		}

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

	t.Run("perf does not write to s3 unless configured", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_CPU_PROFILE_DIR": os.TempDir(),
		}

		config, configErr := NewConfig(testEnv.LookupEnv)
		if configErr != nil {
			ts.Fatalf("failed to initialize perf config: %s", configErr)
		}

		s3Client := &mockS3{}

		perf := NewReporterWithS3(config, s3Client)
		perf.Start()
		defer func() {
			err := perf.Stop()
			// Cleanup the pprof file if we can
			if path.Ext(perf.pprofFileName) == ".pprof" {
				os.Remove(perf.pprofFileName)
			}
			if err != nil {
				ts.Fatalf("error stopping perf: %s", err)
			}
		}()

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

	t.Run("perf doesn't run by default", func(ts *testing.T) {
		// Check that the environment doesn't contain perf configuration
		profileDir, profileDirSet := os.LookupEnv("PERF_CPU_PROFILE_DIR")
		if profileDirSet && profileDir != "" {
			ts.Fatalf(
				"PERF_CPU_PROFILE_DIR env is currently set: %s",
				profileDir,
			)
		}

		bucket, bucketSet := os.LookupEnv("PERF_CPU_S3_BUCKET")
		if bucketSet && bucket != "" {
			ts.Fatalf(
				"PERF_CPU_S3_BUCKET env is currently set: %s",
				bucket,
			)
		}

		perf := NewReporter()
		perf.Start()
		defer func() {
			err := perf.Stop()
			// Cleanup the pprof file if we can
			if path.Ext(perf.pprofFileName) == ".pprof" {
				os.Remove(perf.pprofFileName)
			}
			if err != nil {
				ts.Fatalf("error stopping perf: %s", err)
			}
		}()

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

	t.Run("perf defaults to real s3 client", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_CPU_PROFILE_DIR": os.TempDir(),
			"PERF_CPU_S3_BUCKET":   "ably-logs-dev",
		}

		config, configErr := NewConfig(testEnv.LookupEnv)
		if configErr != nil {
			ts.Fatalf("failed to initialize perf config: %s", configErr)
		}

		perf := NewReporterWithS3(config, nil)

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
