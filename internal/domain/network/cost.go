package network

import (
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// TrafficEstimator converts an edge into a travel-time estimate.
type TrafficEstimator interface {
	Estimate(edge model.Edge, at time.Time) float64
}

// CostCalculator computes time-dependent edge cost from base duration,
// active incidents and an optional traffic estimator.
type CostCalculator struct {
	Traffic TrafficEstimator
}

func NewCostCalculator(traffic TrafficEstimator) *CostCalculator {
	return &CostCalculator{Traffic: traffic}
}

// Seconds returns the travel time for an edge at the given departure time,
// applying active incidents in descending priority order.
func (c *CostCalculator) Seconds(edge model.Edge, req model.RouteRequest, at time.Time, incidents []model.Incident) (float64, bool) {
	base := edge.BaseSeconds
	if c.Traffic != nil {
		if estimate := c.Traffic.Estimate(edge, at); estimate > 0 {
			base = estimate
		}
	}
	factor := 1.0
	for _, incident := range SortedActive(edge.ID, at, incidents) {
		if incident.Closed {
			return 0, false
		}
		if incident.SpeedFactor > 0 {
			factor *= incident.SpeedFactor
		}
	}
	return base / factor, true
}

// SortedActive returns incidents applying to the edge at the time, sorted by
// descending priority and then by ascending creation time.
func SortedActive(edge model.EdgeID, at time.Time, incidents []model.Incident) []model.Incident {
	active := make([]model.Incident, 0)
	for _, incident := range incidents {
		if incident.Applies(edge, at) {
			active = append(active, incident)
		}
	}
	for i := 1; i < len(active); i++ {
		for j := i; j > 0 && incidentFirst(active[j], active[j-1]); j-- {
			active[j], active[j-1] = active[j-1], active[j]
		}
	}
	return active
}

func incidentFirst(a, b model.Incident) bool {
	if a.Priority == b.Priority {
		return a.CreatedAt.Before(b.CreatedAt)
	}
	return a.Priority > b.Priority
}
