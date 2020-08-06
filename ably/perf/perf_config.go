package perf

const _defaultCPUProfileDir = ""
const _defaultCPUProfileS3Bucket = ""
const _defaultHistogramDir = ""
const _defaultHistogramS3Bucket = ""

// Config configures the internal profilers for measuring performance and
// latency information directly from the client
type Config struct {
	CPUProfileDir      string
	CPUProfileS3Bucket string
	HistogramDir       string
	HistogramS3Bucket  string
}

// LookupEnvFunc is an environment lookup function to determin envionment
// configuration
type LookupEnvFunc func(key string) (string, bool)

// NewConfig initializes a perf Config from the supplied environment
func NewConfig(lookupEnv LookupEnvFunc) (*Config, error) {
	config := &Config{
		CPUProfileDir:      _defaultCPUProfileDir,
		CPUProfileS3Bucket: _defaultCPUProfileS3Bucket,
		HistogramDir:       _defaultHistogramDir,
		HistogramS3Bucket:  _defaultHistogramS3Bucket,
	}

	cpuProfile, cpuProfileExists := lookupEnv("PERF_CPU_PROFILE_DIR")
	if cpuProfileExists {
		config.CPUProfileDir = cpuProfile
	}

	cpuProfileS3, cpuProfileS3Exists := lookupEnv("PERF_CPU_PROFILE_S3_BUCKET")
	if cpuProfileS3Exists {
		config.CPUProfileS3Bucket = cpuProfileS3
	}

	histogram, histogramExists := lookupEnv("PERF_HISTOGRAM_DIR")
	if histogramExists {
		config.HistogramDir = histogram
	}

	histogramS3, histogramS3Exists := lookupEnv("PERF_HISTOGRAM_S3_BUCKET")
	if histogramS3Exists {
		config.HistogramS3Bucket = histogramS3
	}

	return config, nil
}
