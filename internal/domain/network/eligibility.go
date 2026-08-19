package network

import (
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// EdgeEligibility evaluates mode, vehicle and access rules for an edge.
type EdgeEligibility struct {
	Incidents []model.Incident
}

func NewEdgeEligibility(incidents []model.Incident) EdgeEligibility {
	return EdgeEligibility{Incidents: append([]model.Incident(nil), incidents...)}
}

// Allowed reports whether the edge can be used for the request at the time.
func (e EdgeEligibility) Allowed(edge model.Edge, req model.RouteRequest, at time.Time) bool {
	if !edge.Supports(req.Mode, at, req.VehicleClass) {
		return false
	}
	if req.AvoidTolls && edge.TollCents > 0 {
		return false
	}
	if req.MaxSlope > 0 {
		if edge.SlopePercent > req.MaxSlope || edge.SlopePercent < -req.MaxSlope {
			return false
		}
	}
	for _, incident := range e.Incidents {
		if incident.Applies(edge.ID, at) && incident.Closed {
			return false
		}
	}
	return true
}

// AllowedByMode lists the modes that may traverse the edge at the given time.
func (e EdgeEligibility) AllowedByMode(edge model.Edge, vehicle string, at time.Time) []model.Mode {
	out := make([]model.Mode, 0, len(edge.Modes))
	for _, mode := range edge.Modes {
		if edge.Supports(mode, at, vehicle) {
			out = append(out, mode)
		}
	}
	return out
}
