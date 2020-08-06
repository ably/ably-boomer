package perf

import (
	"testing"
)

type testEnvMap map[string]string

func (t testEnvMap) LookupEnv(key string) (string, bool) {
	key, ok := t[key]
	return key, ok
}

func TestNewPerfConfig(t *testing.T) {

	t.Run("perf environment defaults", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{}

		config, err := NewConfig(testEnv.LookupEnv)
		if err != nil {
			t.Fatalf("failed to initialize perf config: %s", err)
		}

		if config.CPUProfileDir != _defaultCPUProfileDir {
			t.Errorf(
				"CPUProfileDir was incorrect, got: %s, wanted: %s",
				config.CPUProfileDir,
				_defaultCPUProfileDir,
			)
		}

		if config.CPUProfileS3Bucket != _defaultCPUProfileS3Bucket {
			t.Errorf(
				"CPUProfileS3Bucket was incorrect, got: %s, wanted: %s",
				config.CPUProfileS3Bucket,
				_defaultCPUProfileS3Bucket,
			)
		}

		if config.HistogramDir != _defaultHistogramDir {
			t.Errorf(
				"HistogramDir was incorrect, got: %s, wanted: %s",
				config.HistogramDir,
				_defaultHistogramDir,
			)
		}

		if config.HistogramS3Bucket != _defaultHistogramS3Bucket {
			t.Errorf(
				"HistogramS3Bucket was incorrect, got: %s, wanted: %s",
				config.HistogramS3Bucket,
				_defaultHistogramS3Bucket,
			)
		}
	})

	t.Run("all perf environment variables set", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_CPU_PROFILE_DIR":       "/tmp",
			"PERF_CPU_PROFILE_S3_BUCKET": "ably-logs-dev",
			"PERF_HISTOGRAM_DIR":         "/tmp/histogram",
			"PERF_HISTOGRAM_S3_BUCKET":   "ably-hist-dev",
		}

		config, err := NewConfig(testEnv.LookupEnv)
		if err != nil {
			t.Fatalf("failed to initialize perf config: %s", err)
		}

		if config.CPUProfileDir != testEnv["PERF_CPU_PROFILE_DIR"] {
			t.Errorf(
				"CPUProfileDir was incorrect, got: %s, wanted: %s",
				config.CPUProfileDir,
				testEnv["PERF_CPU_PROFILE_DIR"],
			)
		}

		if config.CPUProfileS3Bucket != testEnv["PERF_CPU_PROFILE_S3_BUCKET"] {
			t.Errorf(
				"CPUProfileS3Bucket was incorrect, got: %s, wanted: %s",
				config.CPUProfileS3Bucket,
				testEnv["PERF_CPU_PROFILE_S3_BUCKET"],
			)
		}

		if config.HistogramDir != testEnv["PERF_HISTOGRAM_DIR"] {
			t.Errorf(
				"HistogramS3Dir was incorrect, got: %s, wanted: %s",
				config.HistogramDir,
				testEnv["PERF_HISTOGRAM_DIR"],
			)
		}

		if config.HistogramS3Bucket != testEnv["PERF_HISTOGRAM_S3_BUCKET"] {
			t.Errorf(
				"HistogramS3Bucket was incorrect, got: %s, wanted: %s",
				config.HistogramDir,
				testEnv["PERF_HISTOGRAM_S3_BUCKET"],
			)
		}
	})
}
