package analytics

import (
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

type EdgeObservation struct {
	EdgeID    model.EdgeID `json:"edgeId"`
	At        time.Time    `json:"at"`
	SpeedKPH  float64      `json:"speedKph"`
	Occupancy float64      `json:"occupancy"`
	Source    string       `json:"source"`
}

func (o EdgeObservation) Validate() error {
	if o.EdgeID == "" || o.At.IsZero() {
		return errors.New("edge and timestamp required")
	}
	if o.SpeedKPH < 0 || o.Occupancy < 0 || o.Occupancy > 1 {
		return errors.New("invalid traffic values")
	}
	return nil
}

type TrafficIndex struct {
	mu     sync.RWMutex
	values map[model.EdgeID][]EdgeObservation
	limit  int
}

func NewTrafficIndex(limit int) *TrafficIndex {
	if limit < 100 {
		limit = 100
	}
	return &TrafficIndex{values: map[model.EdgeID][]EdgeObservation{}, limit: limit}
}
func (i *TrafficIndex) Record(o EdgeObservation) error {
	if err := o.Validate(); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	list := i.values[o.EdgeID]
	list = append(list, o)
	if len(list) > i.limit {
		list = list[len(list)-i.limit:]
	}
	i.values[o.EdgeID] = list
	return nil
}
func (i *TrafficIndex) Latest(edge model.EdgeID, at time.Time) (EdgeObservation, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	list := i.values[edge]
	var found EdgeObservation
	ok := false
	for _, o := range list {
		if !o.At.After(at) && (!ok || o.At.After(found.At)) {
			found = o
			ok = true
		}
	}
	return found, ok
}
func (i *TrafficIndex) Estimate(edge model.Edge, at time.Time) float64 {
	obs, ok := i.Latest(edge.ID, at)
	if !ok || obs.SpeedKPH <= 0 {
		return edge.BaseSeconds
	}
	metersPerSecond := obs.SpeedKPH / 3.6
	return edge.LengthMeters / metersPerSecond
}
func (i *TrafficIndex) Summary(edge model.EdgeID, from, to time.Time) map[string]float64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	var speed, occupancy float64
	var count float64
	for _, o := range i.values[edge] {
		if !o.At.Before(from) && !o.At.After(to) {
			speed += o.SpeedKPH
			occupancy += o.Occupancy
			count++
		}
	}
	if count == 0 {
		return map[string]float64{"samples": 0}
	}
	return map[string]float64{"samples": count, "average_speed_kph": speed / count, "average_occupancy": occupancy / count}
}

type DemandPair struct {
	From  model.NodeID `json:"from"`
	To    model.NodeID `json:"to"`
	Count float64      `json:"count"`
}
type ODMatrix struct {
	mu    sync.RWMutex
	cells map[string]float64
}

func NewODMatrix() *ODMatrix { return &ODMatrix{cells: map[string]float64{}} }
func (m *ODMatrix) Add(from, to model.NodeID, count float64) error {
	if from == "" || to == "" || count < 0 {
		return errors.New("invalid demand pair")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cells[string(from)+"->"+string(to)] += count
	return nil
}
func (m *ODMatrix) Rows() []DemandPair {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]DemandPair, 0, len(m.cells))
	for key, count := range m.cells {
		var from, to string
		for j := 0; j+1 < len(key); j++ {
			if key[j:j+2] == "->" {
				from, to = key[:j], key[j+2:]
				break
			}
		}
		out = append(out, DemandPair{From: model.NodeID(from), To: model.NodeID(to), Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From == out[j].From {
			return out[i].To < out[j].To
		}
		return out[i].From < out[j].From
	})
	return out
}
func (m *ODMatrix) Normalize() map[string]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	totals := map[string]float64{}
	for key, count := range m.cells {
		for j := 0; j+1 < len(key); j++ {
			if key[j:j+2] == "->" {
				totals[key[:j]] += count
				break
			}
		}
	}
	out := map[string]float64{}
	for key, count := range m.cells {
		for j := 0; j+1 < len(key); j++ {
			if key[j:j+2] == "->" {
				total := totals[key[:j]]
				if total > 0 {
					out[key] = count / total
				}
				break
			}
		}
	}
	return out
}

type Reliability struct {
	Samples     int64   `json:"samples"`
	Failures    int64   `json:"failures"`
	MeanSeconds float64 `json:"meanSeconds"`
	P95Seconds  float64 `json:"p95Seconds"`
}

func ComputeReliability(samples []float64, failures int64) Reliability {
	if failures < 0 {
		failures = 0
	}
	clean := make([]float64, 0, len(samples))
	for _, v := range samples {
		if v >= 0 && !math.IsNaN(v) && !math.IsInf(v, 0) {
			clean = append(clean, v)
		}
	}
	sort.Float64s(clean)
	mean := 0.0
	for _, v := range clean {
		mean += v
	}
	if len(clean) > 0 {
		mean /= float64(len(clean))
	}
	p95 := 0.0
	if len(clean) > 0 {
		p95 = clean[int(float64(len(clean)-1)*.95)]
	}
	return Reliability{Samples: int64(len(clean)), Failures: failures, MeanSeconds: mean, P95Seconds: p95}
}
