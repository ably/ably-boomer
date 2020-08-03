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

		if config.CPUProfileDir != defaultCPUProfileDir {
			t.Errorf(
				"CPUProfileDir was incorrect, got: %s, wanted: %s",
				config.CPUProfileDir,
				defaultCPUProfileDir,
			)
		}

		if config.S3Bucket != defaultS3Bucket {
			t.Errorf(
				"S3Bucket was incorrect, got: %s, wanted: %s",
				config.S3Bucket,
				defaultS3Bucket,
			)
		}
	})

	t.Run("all perf environment variables set", func(ts *testing.T) {
		var testEnv testEnvMap = map[string]string{
			"PERF_CPU_PROFILE_DIR": "/tmp",
			"PERF_CPU_S3_BUCKET":   "ably-logs-dev",
		}

		config, err := NewConfig(testEnv.LookupEnv)
		if err != nil {
			t.Fatalf("failed to initialize perf config: %s", err)
		}

		if config.CPUProfileDir != testEnv["PERF_CPU_PROFILE_DIR"] {
			t.Errorf(
				"CPUProfileDir was incorrect, got: %s, wanted: %s",
				config.CPUProfileDir,
				defaultCPUProfileDir,
			)
		}

		if config.S3Bucket != testEnv["PERF_CPU_S3_BUCKET"] {
			t.Errorf(
				"S3Bucket was incorrect, got: %s, wanted: %s",
				config.S3Bucket,
				defaultS3Bucket,
			)
		}

	})
}
