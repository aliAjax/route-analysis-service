package analytics

import (
	"errors"
	"sort"
	"time"
)

type TimeBucket struct {
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	Samples        int64     `json:"samples"`
	AverageSeconds float64   `json:"averageSeconds"`
	P95Seconds     float64   `json:"p95Seconds"`
}

func BuildTimeProfile(values map[time.Time][]float64, window time.Duration) []TimeBucket {
	if window <= 0 {
		window = time.Hour
	}
	starts := make([]time.Time, 0, len(values))
	for start := range values {
		starts = append(starts, start)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i].Before(starts[j]) })
	out := make([]TimeBucket, 0, len(starts))
	for _, start := range starts {
		sample := values[start]
		reliability := ComputeReliability(sample, 0)
		out = append(out, TimeBucket{Start: start, End: start.Add(window), Samples: reliability.Samples, AverageSeconds: reliability.MeanSeconds, P95Seconds: reliability.P95Seconds})
	}
	return out
}
func ValidateTimeProfile(buckets []TimeBucket) error {
	var previous time.Time
	for _, bucket := range buckets {
		if bucket.Start.IsZero() || bucket.End.Before(bucket.Start) {
			return errors.New("invalid time bucket")
		}
		if !previous.IsZero() && bucket.Start.Before(previous) {
			return errors.New("time buckets are not ordered")
		}
		if bucket.Samples < 0 || bucket.AverageSeconds < 0 || bucket.P95Seconds < 0 {
			return errors.New("negative profile metric")
		}
		previous = bucket.End
	}
	return nil
}

func MergeProfiles(profiles ...[]TimeBucket) []TimeBucket {
	merged := map[time.Time][]float64{}
	for _, profile := range profiles {
		for _, bucket := range profile {
			if bucket.Samples > 0 {
				merged[bucket.Start] = append(merged[bucket.Start], bucket.AverageSeconds)
			}
		}
	}
	values := make([]TimeBucket, 0, len(merged))
	for _, bucket := range BuildTimeProfile(merged, time.Hour) {
		values = append(values, bucket)
	}
	return values
}

func Percentile(values []float64, p float64) (float64, bool) {
	if len(values) == 0 || p < 0 || p > 1 {
		return 0, false
	}
	clean := append([]float64(nil), values...)
	sort.Float64s(clean)
	index := int(float64(len(clean)-1) * p)
	return clean[index], true
}

func WeightedAverage(values []float64, weights []float64) (float64, bool) {
	if len(values) == 0 || len(values) != len(weights) {
		return 0, false
	}
	total, weight := 0.0, 0.0
	for i, value := range values {
		if weights[i] < 0 {
			return 0, false
		}
		total += value * weights[i]
		weight += weights[i]
	}
	if weight == 0 {
		return 0, false
	}
	return total / weight, true
}
