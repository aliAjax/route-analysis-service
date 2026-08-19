package incident

import (
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// EligibilitySnapshot is the subset of incident data needed to decide whether
// an edge is closed or slowed at a given departure time.
type EligibilitySnapshot struct {
	Incidents []model.Incident
}

// NewEligibilitySnapshot copies the input slice so callers can mutate their own
// copy without changing the snapshot.
func NewEligibilitySnapshot(incidents []model.Incident) EligibilitySnapshot {
	return EligibilitySnapshot{Incidents: append([]model.Incident(nil), incidents...)}
}

// ActiveFor returns incidents that apply to the edge at the given time, sorted
// by descending priority (higher priority first).
func (s EligibilitySnapshot) ActiveFor(edge model.EdgeID, at time.Time) []model.Incident {
	out := make([]model.Incident, 0)
	for _, item := range s.Incidents {
		if item.Applies(edge, at) {
			out = append(out, item)
		}
	}
	// insertion sort keeps small lists deterministic without importing sort.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Priority > out[j-1].Priority; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// EffectiveSpeedFactor returns the multiplicative speed factor after applying
// every active incident, or 1 when no incident changes speed.
func (s EligibilitySnapshot) EffectiveSpeedFactor(edge model.EdgeID, at time.Time) float64 {
	factor := 1.0
	for _, item := range s.ActiveFor(edge, at) {
		if item.SpeedFactor > 0 {
			factor *= item.SpeedFactor
		}
	}
	return factor
}

// Closed reports whether any active incident closes the edge.
func (s EligibilitySnapshot) Closed(edge model.EdgeID, at time.Time) bool {
	for _, item := range s.ActiveFor(edge, at) {
		if item.Closed {
			return true
		}
	}
	return false
}
