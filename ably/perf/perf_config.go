package perf

const defaultCPUProfileDir = ""
const defaultS3Bucket = ""

// Config configures the internal profilers for measuring performance and
// latency information directly from the client
type Config struct {
	CPUProfileDir string
	S3Bucket      string
}

// LookupEnvFunc is an environment lookup function to determin envionment
// configuration
type LookupEnvFunc func(key string) (string, bool)

// NewConfig initializes a perf Config from the supplied environment
func NewConfig(lookupEnv LookupEnvFunc) (*Config, error) {
	config := &Config{
		CPUProfileDir: defaultCPUProfileDir,
		S3Bucket:      defaultS3Bucket,
	}

	cpuProfile, cpuProfileExists := lookupEnv("PERF_CPU_PROFILE_DIR")
	if cpuProfileExists {
		config.CPUProfileDir = cpuProfile
	}

	s3Bucket, s3BucketExists := lookupEnv("PERF_CPU_S3_BUCKET")
	if s3BucketExists {
		config.S3Bucket = s3Bucket
	}

	return config, nil
}
