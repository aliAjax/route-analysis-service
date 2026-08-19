package incident

import (
	"errors"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// BatchResult records the outcome of a multi-incident lifecycle operation.
type BatchResult struct {
	Updated []model.IncidentID `json:"updated"`
	Failed  []Failure          `json:"failed"`
}

// Failure explains why one incident could not be updated.
type Failure struct {
	Incident model.IncidentID `json:"incident"`
	Reason   string           `json:"reason"`
}

// BatchTransition moves every matching incident to the target status and
// returns per-item results without aborting the whole batch.
func BatchTransition(items []model.Incident, predicate func(model.Incident) bool, next model.IncidentStatus) (updated []model.Incident, result BatchResult) {
	updated = make([]model.Incident, 0, len(items))
	result = BatchResult{Updated: []model.IncidentID{}, Failed: []Failure{}}
	for _, item := range items {
		if predicate != nil && !predicate(item) {
			continue
		}
		nextItem, err := Transition(item, next)
		if err != nil {
			result.Failed = append(result.Failed, Failure{Incident: item.ID, Reason: err.Error()})
			updated = append(updated, item)
			continue
		}
		result.Updated = append(result.Updated, item.ID)
		updated = append(updated, nextItem)
	}
	return updated, result
}

// Expired returns incidents whose window ended before the given instant.
func Expired(items []model.Incident, at time.Time) []model.Incident {
	out := make([]model.Incident, 0)
	for _, item := range items {
		if !item.Window.End.IsZero() && item.Window.End.Before(at) {
			out = append(out, item)
		}
	}
	return out
}

// ResolveExpired moves expired active incidents to resolved and returns the
// changed items together with their IDs.
func ResolveExpired(items []model.Incident, at time.Time) (changed []model.Incident, ids []model.IncidentID, err error) {
	changed = make([]model.Incident, 0)
	ids = make([]model.IncidentID, 0)
	for _, item := range items {
		if item.Status != model.IncidentActive {
			continue
		}
		if !item.Window.End.IsZero() && item.Window.End.Before(at) {
			next, transitionErr := Transition(item, model.IncidentResolved)
			if transitionErr != nil {
				return nil, nil, transitionErr
			}
			changed = append(changed, next)
			ids = append(ids, item.ID)
		}
	}
	return changed, ids, nil
}

var ErrEmptyBatch = errors.New("incident batch is empty")
