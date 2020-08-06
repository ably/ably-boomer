package perf

import (
	"bytes"
	"io"
	"math/rand"
	"os"
	"path"
	"sort"
	"strconv"
	"testing"
)

func TestNewHistogram(t *testing.T) {

	t.Run("NewHistogram provides a 60s default", func(ts *testing.T) {
		hist := NewDefaultHistogram()

		for i := 0; i <= 60001; i++ {
			hist.Add(int64(i))
		}

		expectedPercentiles := percentilesFromArray([]int64{
			0,
			3000,
			6000,
			9000,
			12000,
			15000,
			18000,
			21000,
			24000,
			27000,
			30000,
			33001,
			36001,
			39001,
			42001,
			45001,
			48001,
			51001,
			54001,
			57001,
			59401,
			59941,
			59995,
			60001,
			60002,
		})

		percentiles := hist.Percentiles()

		assertEqualPercentiles(ts, percentiles, expectedPercentiles)
	})

	t.Run("Histogram with bucket spacing works correctly", func(ts *testing.T) {
		// Like the default, but with 5ms buckets instead of 1ms buckets
		hist := NewHistogram(12000, 1, 5)

		for i := 0; i <= 60001; i++ {
			hist.Add(int64(i))
		}

		expectedPercentiles := percentilesFromArray([]int64{
			0,
			3000,
			6000,
			9000,
			12000,
			15000,
			18000,
			21000,
			24000,
			27000,
			30000,
			33005,
			36005,
			39005,
			42005,
			45005,
			48005,
			51005,
			54005,
			57005,
			59405,
			59945,
			59995,
			60001,
			60002,
		})

		percentiles := hist.Percentiles()

		assertEqualPercentiles(ts, percentiles, expectedPercentiles)
	})

	t.Run("Histogram with all samples below min", func(ts *testing.T) {
		hist := NewDefaultHistogram()

		for i := 0; i <= 60001; i++ {
			hist.Add(-1 * int64(i))
		}

		expectedPercentiles := percentilesFromArray([]int64{
			-60001,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			0,
			60002,
		})

		percentiles := hist.Percentiles()

		assertEqualPercentiles(ts, percentiles, expectedPercentiles)
	})

	t.Run("Histogram with all samples above max", func(ts *testing.T) {
		hist := NewDefaultHistogram()

		for i := 0; i <= 60001; i++ {
			hist.Add(60001 + int64(i))
		}

		expectedPercentiles := percentilesFromArray([]int64{
			60001,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			120002,
			60002,
		})

		percentiles := hist.Percentiles()

		assertEqualPercentiles(ts, percentiles, expectedPercentiles)
	})

	t.Run("Empty percentile contains zeroes", func(ts *testing.T) {
		hist := NewDefaultHistogram()
		expectedPercentiles := &Percentiles{}
		percentiles := hist.Percentiles()
		assertEqualPercentiles(ts, percentiles, expectedPercentiles)
	})

	t.Run("Histogram max clamped", func(ts *testing.T) {
		// Like the default, but with 5ms buckets instead of 1ms buckets
		hist := NewHistogram(12000, 1, 5)

		for i := 0; i <= 60001; i++ {
			if i > 59994 {
				hist.Add(59994)
			} else {
				hist.Add(int64(i))
			}
		}

		expectedPercentiles := percentilesFromArray([]int64{
			0,
			3000,
			6000,
			9000,
			12000,
			15000,
			18000,
			21000,
			24000,
			27000,
			30000,
			33005,
			36005,
			39005,
			42005,
			45005,
			48005,
			51005,
			54005,
			57005,
			59405,
			59945,
			59994,
			59994,
			60002,
		})

		percentiles := hist.Percentiles()

		assertEqualPercentiles(ts, percentiles, expectedPercentiles)
	})

	t.Run("Histogram min values max clamped", func(ts *testing.T) {
		// Like the default, but with 5ms buckets instead of 1ms buckets
		hist := NewHistogram(4, 1, 5)

		hist.Add(-6)
		hist.Add(-12)

		expectedPercentiles := percentilesFromArray([]int64{
			-12,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			-6,
			2,
		})

		percentiles := hist.Percentiles()

		assertEqualPercentiles(ts, percentiles, expectedPercentiles)
	})

	t.Run("Histogram with negative samples", func(ts *testing.T) {
		hist := NewHistogram(60000, -60000, 1)

		for i := 0; i <= 60001; i++ {
			hist.Add(-1 * int64(i))
		}

		expectedPercentiles := percentilesFromArray([]int64{
			-60001,
			-57001,
			-54001,
			-51001,
			-48001,
			-45001,
			-42001,
			-39001,
			-36001,
			-33001,
			-30001,
			-27000,
			-24000,
			-21000,
			-18000,
			-15000,
			-12000,
			-9000,
			-6000,
			-3000,
			-600,
			-60,
			-6,
			0,
			60002,
		})

		percentiles := hist.Percentiles()

		assertEqualPercentiles(ts, percentiles, expectedPercentiles)
	})

	t.Run("Histogram with negative width", func(ts *testing.T) {
		hist := NewHistogram(12000, 60000, -5)

		for i := 0; i <= 60001; i++ {
			hist.Add(int64(i))
		}

		expectedPercentiles := percentilesFromArray([]int64{
			0,
			3000,
			6000,
			9000,
			12000,
			15000,
			18000,
			21000,
			24000,
			27000,
			30000,
			33005,
			36005,
			39005,
			42005,
			45005,
			48005,
			51005,
			54005,
			57005,
			59405,
			59945,
			59995,
			60001,
			60002,
		})

		percentiles := hist.Percentiles()

		assertEqualPercentiles(ts, percentiles, expectedPercentiles)
	})
}

func TestNewHistogramSimulation(t *testing.T) {
	t.Run("NewHistogram provides a 60s default", func(ts *testing.T) {
		hist := NewDefaultHistogram()
		samples := int64(1000000)

		sampleLog := make([]int64, 0, samples)

		for i := int64(0); i <= samples; i++ {
			sample := int64(rand.Intn(60001))
			hist.Add(sample)
			sampleLog = append(sampleLog, sample)
		}

		sort.Slice(sampleLog, func(i, j int) bool {
			return sampleLog[i] < sampleLog[j]
		})

		expectedPercentiles := percentilesFromArray([]int64{
			sampleLog[0],
			sampleLog[percentileToSamples(samples, 5, 100)],
			sampleLog[percentileToSamples(samples, 10, 100)],
			sampleLog[percentileToSamples(samples, 15, 100)],
			sampleLog[percentileToSamples(samples, 20, 100)],
			sampleLog[percentileToSamples(samples, 25, 100)],
			sampleLog[percentileToSamples(samples, 30, 100)],
			sampleLog[percentileToSamples(samples, 35, 100)],
			sampleLog[percentileToSamples(samples, 40, 100)],
			sampleLog[percentileToSamples(samples, 45, 100)],
			sampleLog[percentileToSamples(samples, 50, 100)],
			sampleLog[percentileToSamples(samples, 55, 100)],
			sampleLog[percentileToSamples(samples, 60, 100)],
			sampleLog[percentileToSamples(samples, 65, 100)],
			sampleLog[percentileToSamples(samples, 70, 100)],
			sampleLog[percentileToSamples(samples, 75, 100)],
			sampleLog[percentileToSamples(samples, 80, 100)],
			sampleLog[percentileToSamples(samples, 85, 100)],
			sampleLog[percentileToSamples(samples, 90, 100)],
			sampleLog[percentileToSamples(samples, 95, 100)],
			sampleLog[percentileToSamples(samples, 99, 100)],
			sampleLog[percentileToSamples(samples, 999, 1000)],
			sampleLog[percentileToSamples(samples, 9999, 10000)],
			sampleLog[samples-1],
			int64(len(sampleLog)),
		})

		percentiles := hist.Percentiles()

		assertEqualPercentiles(ts, percentiles, expectedPercentiles)
	})
}

func TestHistogramEncoding(t *testing.T) {
	t.Run("Histogram encodes to binary", func(ts *testing.T) {
		var buf bytes.Buffer
		expectedHistogramCount := 10
		origHistograms := make([]*Histogram, 0, expectedHistogramCount)

		writer := NewHistogramWriter(&buf)
		writeNilErr := writer.Write("", nil)

		if writeNilErr == nil {
			ts.Errorf("expected writer.write(nil) to return error")
		}

		for i := 0; i < expectedHistogramCount; i++ {
			histogram := NewDefaultHistogram()
			origHistograms = append(origHistograms, histogram)

			for i := 0; i <= 1000000; i++ {
				histogram.Add(int64(rand.Intn(60001)))
			}

			id := strconv.Itoa(i)
			writeErr := writer.Write(id, histogram)
			if writeErr != nil {
				ts.Errorf("unexpected error encoding histogram: %s", writeErr)
			}
		}

		reader := NewHistogramReader(&buf)

		histogramCount := 0
		for {
			expectedID := strconv.Itoa(histogramCount)
			id, histogram, err := reader.Read()

			if err == io.EOF {
				break
			} else if err != nil {
				ts.Errorf("unexpected error decoding histogram: %s", err)
				break
			}

			if id != expectedID {
				ts.Errorf(
					"unecpected histogram id, got %s, wanted %s",
					id,
					expectedID,
				)
			}

			if histogramCount < expectedHistogramCount {
				assertEqualHistograms(
					ts,
					histogram,
					origHistograms[histogramCount],
				)
				assertEqualPercentiles(
					ts,
					histogram.Percentiles(),
					origHistograms[histogramCount].Percentiles(),
				)
				histogramCount++
			}
		}

		if histogramCount != expectedHistogramCount {
			ts.Errorf(
				"error reading histogram, wanted: %d, got: %d",
				histogramCount,
				expectedHistogramCount,
			)
		}
	})
}

func assertEqualHistogramFile(
	t *testing.T,
	fileName string,
	expectedHistMap map[string]*Histogram,
) {
	fileExt := path.Ext(fileName)
	expectedFileExt := ".hist"
	if fileExt != expectedFileExt {
		t.Fatalf(
			"unexpected histogram extension, got: %s, wanted: %s",
			fileExt,
			expectedFileExt,
		)
	}
	histStat, histStatErr := os.Stat(fileName)
	if histStatErr != nil {
		t.Fatalf(
			"histogram file missing from disk: %s",
			histStatErr,
		)
	} else if histStat.Size() == 0 {
		t.Fatalf("histogram file is empty")
	}

	histFile, histFileErr := os.Open(fileName)
	if histFileErr != nil {
		t.Fatalf(
			"unexpected error opening histogram file: %s",
			histFileErr,
		)
	}

	reader := NewHistogramReader(histFile)

	histMap := map[string]*Histogram{}
	for {
		id, histogram, err := reader.Read()

		if err == io.EOF {
			break
		} else if err != nil {
			t.Errorf("unexpected error decoding histogram: %s", err)
			break
		}

		_, histExists := histMap[id]
		if histExists {
			t.Errorf("duplicate histogram found in stream: %s", id)
		} else {
			histMap[id] = histogram
		}
	}

	if len(histMap) != len(expectedHistMap) {
		t.Errorf(
			"unexpected histogram count, wanted: %d, got: %d",
			len(expectedHistMap),
			len(histMap),
		)
	} else {
		for key, hist := range histMap {
			assertEqualHistograms(t, hist, expectedHistMap[key])
		}
	}
}

func assertEqualHistograms(
	t *testing.T,
	actual *Histogram,
	expected *Histogram,
) {
	if actual == nil || expected == nil {
		if actual == expected {
			return
		}

		if actual == nil {
			t.Errorf("expected non-nil histogram")
		} else {
			t.Errorf("expected nil histogram")
		}

		return
	}

	if actual.bucketCount != expected.bucketCount {
		t.Errorf(
			"unepxected histogram bucketCount value, wanted: %d, got %d",
			expected.bucketCount,
			actual.bucketCount,
		)
	}

	if len(actual.buckets) != len(expected.buckets) {
		t.Errorf(
			"unpected histogram buckets length, wanted: %d, got %d",
			len(expected.buckets),
			len(actual.buckets),
		)
	} else {
		for i, expectedCount := range expected.buckets {
			if actual.buckets[i] != expectedCount {
				t.Errorf(
					"unexpected bucket %d sample count, wanted %d, got: %d",
					i,
					expectedCount,
					actual.buckets[i],
				)
				break
			}
		}
	}

	if actual.min != expected.min {
		t.Errorf(
			"unepxected histogram min value, wanted: %d, got %d",
			expected.min,
			actual.min,
		)
	}

	if actual.max != expected.max {
		t.Errorf(
			"unepxected histogram max value, wanted: %d, got %d",
			expected.max,
			actual.max,
		)
	}

	if actual.bucketWidth != expected.bucketWidth {
		t.Errorf(
			"unepxected histogram bucketWidth value, wanted: %d, got %d",
			expected.bucketWidth,
			actual.bucketWidth,
		)
	}

	if actual.sampleMin != expected.sampleMin {
		t.Errorf(
			"unepxected histogram bucketWidth value, wanted: %d, got %d",
			expected.sampleMin,
			actual.sampleMin,
		)
	}

	if actual.sampleMax != expected.sampleMax {
		t.Errorf(
			"unepxected histogram sampleMax value, wanted: %d, got %d",
			expected.sampleMax,
			actual.sampleMax,
		)
	}

	if actual.lowSampleCount != expected.lowSampleCount {
		t.Errorf(
			"unepxected histogram lowSampleCount value, wanted: %d, got %d",
			expected.lowSampleCount,
			actual.lowSampleCount,
		)
	}

	if actual.highSampleCount != expected.highSampleCount {
		t.Errorf(
			"unepxected histogram highSampleCount value, wanted: %d, got %d",
			expected.highSampleCount,
			actual.highSampleCount,
		)
	}

	if actual.totalSamples != expected.totalSamples {
		t.Errorf(
			"unepxected histogram totalSamples value, wanted: %d, got %d",
			expected.totalSamples,
			actual.totalSamples,
		)
	}
}

func assertEqualPercentiles(
	t *testing.T,
	actual *Percentiles,
	expected *Percentiles,
) {
	if actual == nil || expected == nil {
		if actual == expected {
			return
		}

		if actual == nil {
			t.Errorf("expected non-nil percentile")
		} else {
			t.Errorf("expected nil percentile")
		}

		return
	}

	if actual.Min != expected.Min {
		t.Errorf(
			"unepxected percentile Max value, wanted: %d, got %d",
			expected.Max,
			actual.Max,
		)
	}

	if actual.Pct5 != expected.Pct5 {
		t.Errorf(
			"unepxected percentile Pct5 value, wanted: %d, got %d",
			expected.Pct5,
			actual.Pct5,
		)
	}

	if actual.Pct10 != expected.Pct10 {
		t.Errorf(
			"unepxected percentile Pct10 value, wanted: %d, got %d",
			expected.Pct10,
			actual.Pct10,
		)
	}

	if actual.Pct15 != expected.Pct15 {
		t.Errorf(
			"unepxected percentile Pct15 value, wanted: %d, got %d",
			expected.Pct15,
			actual.Pct15,
		)
	}

	if actual.Pct20 != expected.Pct20 {
		t.Errorf(
			"unepxected percentile Pct20 value, wanted: %d, got %d",
			expected.Pct20,
			actual.Pct20,
		)
	}

	if actual.Pct25 != expected.Pct25 {
		t.Errorf(
			"unepxected percentile Pct25 value, wanted: %d, got %d",
			expected.Pct25,
			actual.Pct25,
		)
	}

	if actual.Pct30 != expected.Pct30 {
		t.Errorf(
			"unepxected percentile Pct30 value, wanted: %d, got %d",
			expected.Pct30,
			actual.Pct30,
		)
	}

	if actual.Pct35 != expected.Pct35 {
		t.Errorf(
			"unepxected percentile Pct35 value, wanted: %d, got %d",
			expected.Pct35,
			actual.Pct35,
		)
	}

	if actual.Pct40 != expected.Pct40 {
		t.Errorf(
			"unepxected percentile Pct40 value, wanted: %d, got %d",
			expected.Pct40,
			actual.Pct40,
		)
	}

	if actual.Pct45 != expected.Pct45 {
		t.Errorf(
			"unepxected percentile Pct45 value, wanted: %d, got %d",
			expected.Pct45,
			actual.Pct45,
		)
	}

	if actual.Pct50 != expected.Pct50 {
		t.Errorf(
			"unepxected percentile Pct50 value, wanted: %d, got %d",
			expected.Pct50,
			actual.Pct50,
		)
	}

	if actual.Pct55 != expected.Pct55 {
		t.Errorf(
			"unepxected percentile Pct55 value, wanted: %d, got %d",
			expected.Pct55,
			actual.Pct55,
		)
	}

	if actual.Pct60 != expected.Pct60 {
		t.Errorf(
			"unepxected percentile Pct60 value, wanted: %d, got %d",
			expected.Pct60,
			actual.Pct60,
		)
	}

	if actual.Pct65 != expected.Pct65 {
		t.Errorf(
			"unepxected percentile Pct65 value, wanted: %d, got %d",
			expected.Pct65,
			actual.Pct65,
		)
	}

	if actual.Pct70 != expected.Pct70 {
		t.Errorf(
			"unepxected percentile Pct70 value, wanted: %d, got %d",
			expected.Pct70,
			actual.Pct70,
		)
	}

	if actual.Pct75 != expected.Pct75 {
		t.Errorf(
			"unepxected percentile Pct75 value, wanted: %d, got %d",
			expected.Pct75,
			actual.Pct75,
		)
	}

	if actual.Pct80 != expected.Pct80 {
		t.Errorf(
			"unepxected percentile Pct80 value, wanted: %d, got %d",
			expected.Pct80,
			actual.Pct80,
		)
	}

	if actual.Pct85 != expected.Pct85 {
		t.Errorf(
			"unepxected percentile Pct85 value, wanted: %d, got %d",
			expected.Pct85,
			actual.Pct85,
		)
	}

	if actual.Pct90 != expected.Pct90 {
		t.Errorf(
			"unepxected percentile Pct90 value, wanted: %d, got %d",
			expected.Pct90,
			actual.Pct90,
		)
	}

	if actual.Pct95 != expected.Pct95 {
		t.Errorf(
			"unepxected percentile 95 value, wanted: %d, got %d",
			expected.Pct95,
			actual.Pct95,
		)
	}

	if actual.Pct99 != expected.Pct99 {
		t.Errorf(
			"unepxected percentile Pct99 value, wanted: %d, got %d",
			expected.Pct99,
			actual.Pct99,
		)
	}

	if actual.Pct999 != expected.Pct999 {
		t.Errorf(
			"unepxected percentile Pct999 value, wanted: %d, got %d",
			expected.Pct999,
			actual.Pct999,
		)
	}

	if actual.Pct9999 != expected.Pct9999 {
		t.Errorf(
			"unepxected percentile Pct9999 value, wanted: %d, got %d",
			expected.Pct9999,
			actual.Pct9999,
		)
	}

	if actual.Max != expected.Max {
		t.Errorf(
			"unepxected percentile Max value, wanted: %d, got %d",
			expected.Max,
			actual.Max,
		)
	}

	if actual.TotalSamples != expected.TotalSamples {
		t.Errorf(
			"unepxected percentile TotalSamples value, wanted: %d, got %d",
			expected.TotalSamples,
			actual.TotalSamples,
		)
	}

}

func percentilesFromArray(arr []int64) *Percentiles {
	return &Percentiles{
		Min:          arr[0],
		Pct5:         arr[1],
		Pct10:        arr[2],
		Pct15:        arr[3],
		Pct20:        arr[4],
		Pct25:        arr[5],
		Pct30:        arr[6],
		Pct35:        arr[7],
		Pct40:        arr[8],
		Pct45:        arr[9],
		Pct50:        arr[10],
		Pct55:        arr[11],
		Pct60:        arr[12],
		Pct65:        arr[13],
		Pct70:        arr[14],
		Pct75:        arr[15],
		Pct80:        arr[16],
		Pct85:        arr[17],
		Pct90:        arr[18],
		Pct95:        arr[19],
		Pct99:        arr[20],
		Pct999:       arr[21],
		Pct9999:      arr[22],
		Max:          arr[23],
		TotalSamples: arr[24],
	}

}
