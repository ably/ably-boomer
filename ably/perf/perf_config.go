package perf

import "github.com/urfave/cli/v2"

// Config configures the internal profilers for measuring performance and
// latency information directly from the client
type Config struct {
	CPUProfileDir string
	S3Bucket      string
}

func DefaultConfig() *Config {
	return &Config{
		CPUProfileDir: "",
		S3Bucket:      "",
	}
}

func (c *Config) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.PathFlag{
			Name:        "cpu-profile-dir",
			Value:       c.CPUProfileDir,
			Destination: &c.CPUProfileDir,
			EnvVars:     []string{"PERF_CPU_PROFILE_DIR"},
			Usage:       "The directory path to write the pprof cpu profile.",
		},
		&cli.StringFlag{
			Name:        "s3-bucket",
			Value:       c.S3Bucket,
			Destination: &c.S3Bucket,
			EnvVars:     []string{"PERF_CPU_S3_BUCKET"},
			Usage:       "The name of the s3 bucket to upload pprof data to.",
		},
	}
}
