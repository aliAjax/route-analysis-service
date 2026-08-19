package analytics

import (
	"errors"
	"math"
	"sort"
	"time"
)

type ForecastPoint struct {
	At    time.Time `json:"at"`
	Value float64   `json:"value"`
	Lower float64   `json:"lower"`
	Upper float64   `json:"upper"`
}

func Forecast(values []float64, horizon int, step time.Duration) ([]ForecastPoint, error) {
	if len(values) < 2 || horizon < 1 || step <= 0 {
		return nil, errors.New("insufficient forecast inputs")
	}
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, errors.New("values must be finite")
		}
	}
	recent := values[len(values)-minInt(5, len(values)):]
	slope := 0.0
	for i := 1; i < len(recent); i++ {
		slope += recent[i] - recent[i-1]
	}
	slope /= float64(len(recent) - 1)
	last := recent[len(recent)-1]
	out := make([]ForecastPoint, 0, horizon)
	base := time.Now().UTC()
	spread := math.Abs(slope)*2 + 1
	for i := 1; i <= horizon; i++ {
		value := last + slope*float64(i)
		out = append(out, ForecastPoint{At: base.Add(time.Duration(i) * step), Value: value, Lower: value - spread, Upper: value + spread})
	}
	return out, nil
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func Smooth(values []float64, window int) []float64 {
	if window < 1 {
		window = 1
	}
	out := make([]float64, len(values))
	for i := range values {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		sum := 0.0
		for j := start; j <= i; j++ {
			sum += values[j]
		}
		out[i] = sum / float64(i-start+1)
	}
	return out
}
func DetectSpikes(values []float64, threshold float64) []int {
	if len(values) < 3 {
		return nil
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	median := sorted[len(sorted)/2]
	spikes := []int{}
	for i, v := range values {
		if median != 0 && math.Abs(v-median)/math.Abs(median) > threshold {
			spikes = append(spikes, i)
		}
	}
	return spikes
}
