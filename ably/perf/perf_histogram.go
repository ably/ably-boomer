package perf

import (
	"encoding/gob"
	"fmt"
	"io"
)

// Default histogram to [1,60000] milliseconds with 1ms buckets
const _defaultMinTime = 1
const _defaultBucketCount = 60000
const _defaultBucketWidth = 1

// Histogram provides a linearly spaced histogram of int64 samples, with
// configurable minimum value and bucket width.
type Histogram struct {
	bucketCount     int
	buckets         []int64
	min             int64
	max             int64
	bucketWidth     int64
	sampleMin       int64
	sampleMax       int64
	lowSampleCount  int64
	highSampleCount int64
	totalSamples    int64
}

// Percentiles contains the different percentiles that repreent the makeup of
// a histogram. The value is not interpolated, but the value of the bucket
// assocatied with the sample that overlaps the percentile line. For example
// if there are 3 samples, the p50 will be the value will be the bucket that
// contains the second sample. If there are 4 samples, the p50 will be the
// value of the bucket that contains the 2nd sample.
//
// Bucket values are reported to be the maximum possible value for that bucket.
// So for example, if we have 4 buckets with a min of 1 and a width of 5,
// the value of bucket 0 will be 5, as samples in bucket 0 can be 1,2,3,4,5.
// This allows us to report that every sample at a given percentile is less
// than or equal to the reported value. If we were to take a bucket midpoint,
// some samples within the percentile could be higher than what is reporterd.
//
// The values are also clamped at the sample max value, so if the histogram is
// formed from 4 buckes with min 1 and width of 5, and the maximum sample
// value is 17, all percentile values will be capped to 17 as this is a better
// upper limit than the bucket max value of 20.
type Percentiles struct {
	Min     int64
	Pct5    int64
	Pct10   int64
	Pct15   int64
	Pct20   int64
	Pct25   int64
	Pct30   int64
	Pct35   int64
	Pct40   int64
	Pct45   int64
	Pct50   int64
	Pct55   int64
	Pct60   int64
	Pct65   int64
	Pct70   int64
	Pct75   int64
	Pct80   int64
	Pct85   int64
	Pct90   int64
	Pct95   int64
	Pct99   int64
	Pct999  int64
	Pct9999 int64
	Max     int64
}

// NewDefaultHistogram returns  a histogram of 1ms to 60000ms, spaced 1ms apart
// (record every ms up 60s)
func NewDefaultHistogram() *Histogram {
	return NewHistogram(
		_defaultBucketCount,
		_defaultMinTime,
		_defaultBucketWidth,
	)
}

// NewHistogram returns a linearly spaced histogram with the given number
// of buckets, starting from the provided min value and spread with the
// specified bucket width
func NewHistogram(bucketCount int, minValue int64, bucketWidth int64) *Histogram {
	if bucketWidth < 0 {
		bucketWidth = -1 * bucketWidth
		minValue = minValue - (bucketWidth * int64(bucketCount)) + 1
	}

	histogram := &Histogram{
		bucketCount: bucketCount,
		buckets:     make([]int64, bucketCount),
		min:         minValue,
		bucketWidth: bucketWidth,
	}

	histogram.max = histogram.maxPossibleValue()

	return histogram
}

// Add appends a sample to the histogram. Values that our out of range are
// tallied in the low/high sample count fields
func (h *Histogram) Add(value int64) {
	if h.totalSamples == 0 {
		h.sampleMin = value
		h.sampleMax = value
	} else {
		if value < h.sampleMin {
			h.sampleMin = value
		}
		if value > h.sampleMax {
			h.sampleMax = value
		}
	}
	h.totalSamples++

	bucket := h.sampleToBucket(value)

	if bucket < 0 {
		h.lowSampleCount++
	} else if bucket >= h.bucketCount {
		h.highSampleCount++
	} else {
		h.buckets[bucket]++
	}
}

// Percentiles returns the histogram percentiles. The statistic means that
// x% of samples were this value or less. When x% refers to a fractional number
// of samples, the ceil(x%) is used. This ensures that the provded values are
// always an upper bound for the given percentile. For example, the p50 of a
// 11 samples is at 5.5 samples. The returned p50 will be the max possible
// value of the bucket that sample 6 lies in. This is then clamped by the
// absolute sample maximum, as the max possible sample can exceed the max
// observed sample.
func (h *Histogram) Percentiles() *Percentiles {
	totalSamples := h.totalSamples
	buckets := h.buckets
	bucketCount := h.bucketCount

	if totalSamples <= 0 {
		return &Percentiles{}
	}

	stob := []int64{
		percentileToSamples(totalSamples, 5, 100),
		percentileToSamples(totalSamples, 10, 100),
		percentileToSamples(totalSamples, 15, 100),
		percentileToSamples(totalSamples, 20, 100),
		percentileToSamples(totalSamples, 25, 100),
		percentileToSamples(totalSamples, 30, 100),
		percentileToSamples(totalSamples, 35, 100),
		percentileToSamples(totalSamples, 40, 100),
		percentileToSamples(totalSamples, 45, 100),
		percentileToSamples(totalSamples, 50, 100),
		percentileToSamples(totalSamples, 55, 100),
		percentileToSamples(totalSamples, 60, 100),
		percentileToSamples(totalSamples, 65, 100),
		percentileToSamples(totalSamples, 70, 100),
		percentileToSamples(totalSamples, 75, 100),
		percentileToSamples(totalSamples, 80, 100),
		percentileToSamples(totalSamples, 85, 100),
		percentileToSamples(totalSamples, 90, 100),
		percentileToSamples(totalSamples, 95, 100),
		percentileToSamples(totalSamples, 99, 100),
		percentileToSamples(totalSamples, 999, 1000),
		percentileToSamples(totalSamples, 9999, 10000),
	}

	btos := make([]int, len(stob))

	var sampleCount int64
	find := 0
	for i := -1; i <= bucketCount; i++ {
		if i < 0 {
			sampleCount += h.lowSampleCount
		} else if i >= bucketCount {
			sampleCount += h.highSampleCount
		} else {
			sampleCount += buckets[i]
		}

		// Determines if this bucket representes one of the percentiles
		for ; find < len(stob) && stob[find] <= sampleCount; find++ {
			btos[find] = i
		}
	}

	return &Percentiles{
		Min:     h.sampleMin,
		Pct5:    h.bucketToMaxValue(btos[0]),
		Pct10:   h.bucketToMaxValue(btos[1]),
		Pct15:   h.bucketToMaxValue(btos[2]),
		Pct20:   h.bucketToMaxValue(btos[3]),
		Pct25:   h.bucketToMaxValue(btos[4]),
		Pct30:   h.bucketToMaxValue(btos[5]),
		Pct35:   h.bucketToMaxValue(btos[6]),
		Pct40:   h.bucketToMaxValue(btos[7]),
		Pct45:   h.bucketToMaxValue(btos[8]),
		Pct50:   h.bucketToMaxValue(btos[9]),
		Pct55:   h.bucketToMaxValue(btos[10]),
		Pct60:   h.bucketToMaxValue(btos[11]),
		Pct65:   h.bucketToMaxValue(btos[12]),
		Pct70:   h.bucketToMaxValue(btos[13]),
		Pct75:   h.bucketToMaxValue(btos[14]),
		Pct80:   h.bucketToMaxValue(btos[15]),
		Pct85:   h.bucketToMaxValue(btos[16]),
		Pct90:   h.bucketToMaxValue(btos[17]),
		Pct95:   h.bucketToMaxValue(btos[18]),
		Pct99:   h.bucketToMaxValue(btos[19]),
		Pct999:  h.bucketToMaxValue(btos[20]),
		Pct9999: h.bucketToMaxValue(btos[21]),
		Max:     h.sampleMax,
	}
}

// Given a sample, return which bucket it belongs to.
func (h *Histogram) sampleToBucket(sample int64) int {
	if sample > h.max {
		return h.bucketCount
	}

	if sample < h.min {
		return -1
	}

	return int((sample - h.min) / h.bucketWidth)
}

// Given a bucket index, compute the maximum value of the bucket and clamp to
// the max sample value
func (h *Histogram) bucketToMaxValue(bucketIndex int) int64 {
	if bucketIndex < 0 {
		possibleMin := h.min - 1
		if possibleMin > h.sampleMax {
			return h.sampleMax
		}
		return possibleMin
	}

	if bucketIndex >= h.bucketCount {
		return h.sampleMax
	}

	bucketMax := h.min + ((1 + int64(bucketIndex)) * h.bucketWidth) - 1

	if h.sampleMax < bucketMax {
		return h.sampleMax
	}

	return bucketMax
}

// Compute the largest possible value in the last bucket
func (h *Histogram) maxPossibleValue() int64 {
	return h.min + ((int64(h.bucketCount)) * h.bucketWidth) - 1
}

// Given a percentile (numerator/denominator), return which sample number
// overlaps the percentile. At overlapping boundaries, round down, i.e. 50%
// of 4 samples is 2 samples (where the percentile line runs between 2 and 3).
func percentileToSamples(totalSamples, numerator, denominator int64) int64 {
	return 1 + (((totalSamples * numerator) - 1) / denominator)
}

// serializedHistogram provides public serializable fields for all fields
// on a histogram. This allows us to serialize internal properties that
// should be private.
type serializedHistogram struct {
	BucketCount     int
	Buckets         []int64
	Min             int64
	Max             int64
	BucketWidth     int64
	SampleMin       int64
	SampleMax       int64
	LowSampleCount  int64
	HighSampleCount int64
	TotalSamples    int64
}

// coverts a histogram to a serialized histogram
func newSerializedHistogram(h *Histogram) *serializedHistogram {
	return &serializedHistogram{
		BucketCount:     h.bucketCount,
		Buckets:         h.buckets,
		Min:             h.min,
		Max:             h.max,
		BucketWidth:     h.bucketWidth,
		SampleMin:       h.sampleMin,
		SampleMax:       h.sampleMax,
		LowSampleCount:  h.lowSampleCount,
		HighSampleCount: h.highSampleCount,
		TotalSamples:    h.totalSamples,
	}
}

// converts a serialized histogram to a histogram
func (s *serializedHistogram) toHistogram() *Histogram {
	return &Histogram{
		bucketCount:     s.BucketCount,
		buckets:         s.Buckets,
		min:             s.Min,
		max:             s.Max,
		bucketWidth:     s.BucketWidth,
		sampleMin:       s.SampleMin,
		sampleMax:       s.SampleMax,
		lowSampleCount:  s.LowSampleCount,
		highSampleCount: s.HighSampleCount,
		totalSamples:    s.TotalSamples,
	}
}

// HistogramReader provides a reader for a stream of encoded histograms
type HistogramReader struct {
	decoder *gob.Decoder
}

// NewHistogramReader creates a histogram reader from the provided io.Reader
func NewHistogramReader(r io.Reader) *HistogramReader {
	return &HistogramReader{
		decoder: gob.NewDecoder(r),
	}
}

// Read returns the next histogram in the histogram stream. The stream will
// return the io.EOF error when the stream has ended.
func (h *HistogramReader) Read() (*Histogram, error) {
	var histogram serializedHistogram
	err := h.decoder.Decode(&histogram)

	if err != nil {
		return nil, err
	}

	return histogram.toHistogram(), nil
}

// HistogramWriter provides a writer for a stream of histograms
type HistogramWriter struct {
	encoder *gob.Encoder
}

// NewHistogramWriter creates a histogram writer from the provided io.Writer
func NewHistogramWriter(w io.Writer) *HistogramWriter {
	return &HistogramWriter{
		encoder: gob.NewEncoder(w),
	}
}

// Write encodes the provided histogram to the underlying io.Writer
func (h *HistogramWriter) Write(histogram *Histogram) error {
	if histogram == nil {
		return fmt.Errorf("cannot encode nil histogram")
	}
	return h.encoder.Encode(newSerializedHistogram(histogram))
}
