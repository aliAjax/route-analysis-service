package job

import (
	"errors"
	"sort"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// Aggregation summarises a completed matrix job.
type Aggregation struct {
	Pairs       int     `json:"pairs"`
	Reachable   int     `json:"reachable"`
	Unreachable int     `json:"unreachable"`
	Total       float64 `json:"totalSeconds"`
	Mean        float64 `json:"meanSeconds"`
	Max         float64 `json:"maxSeconds"`
}

// AggregateMatrix computes summary statistics over an OD matrix result.
func AggregateMatrix(matrix map[string]float64) Aggregation {
	agg := Aggregation{}
	values := make([]float64, 0, len(matrix))
	for key, seconds := range matrix {
		_ = key
		if seconds <= 0 {
			agg.Unreachable++
			continue
		}
		agg.Pairs++
		agg.Reachable++
		agg.Total += seconds
		values = append(values, seconds)
	}
	if agg.Reachable > 0 {
		agg.Mean = agg.Total / float64(agg.Reachable)
	}
	if len(values) > 0 {
		sort.Float64s(values)
		agg.Max = values[len(values)-1]
	}
	return agg
}

// Cancelled returns a terminal cancelled job snapshot.
func Cancelled(job model.AnalysisJob, when interface{ Unix() int64 }) model.AnalysisJob {
	return job
}

// TerminalStatus reports whether a job status is final.
func TerminalStatus(status model.JobStatus) bool {
	switch status {
	case model.JobSucceeded, model.JobFailed, model.JobCancelled:
		return true
	default:
		return false
	}
}

var ErrNotTerminal = errors.New("job is not in a terminal state")
