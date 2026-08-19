package application

import (
	"context"
	"time"

	"github.com/example/route-analysis-service/internal/domain/incident"
	"github.com/example/route-analysis-service/internal/domain/model"
)

// ListIncidentsSorted returns a dataset's incidents ordered by priority.
func (s *Service) ListIncidentsSorted(ctx context.Context, id model.DatasetID) ([]model.Incident, error) {
	items, err := s.repo.ListIncidents(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.resolver.SortByPriority(items), nil
}

// IncidentConflicts reports priority conflicts among active incidents.
func (s *Service) IncidentConflicts(ctx context.Context, id model.DatasetID, at time.Time) ([]incident.Conflict, error) {
	items, err := s.repo.ListIncidents(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.resolver.Conflicts(items, at), nil
}

// ResolveExpiredIncidents moves expired active incidents to resolved and
// persists each change back through the repository.
func (s *Service) ResolveExpiredIncidents(ctx context.Context, id model.DatasetID, at time.Time) ([]model.IncidentID, error) {
	items, err := s.repo.ListIncidents(ctx, id)
	if err != nil {
		return nil, err
	}
	changed, ids, err := incident.ResolveExpired(items, at)
	if err != nil {
		return nil, err
	}
	for _, item := range changed {
		if err := s.repo.UpdateIncident(ctx, item, item.Version); err != nil {
			return ids, err
		}
	}
	return ids, nil
}
