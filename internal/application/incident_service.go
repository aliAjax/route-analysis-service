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
func (s *Service) ResolveExpiredIncidents(ctx context.Context, id model.DatasetID, at time.Time) (ids []model.IncidentID, err error) {
	items, listErr := s.repo.ListIncidents(ctx, id)
	if listErr != nil {
		return nil, listErr
	}
	changed, resolvedIDs, resolveErr := incident.ResolveExpired(items, at)
	if resolveErr != nil {
		return nil, resolveErr
	}
	ids = resolvedIDs
	defer func() { ids = append(ids, resolvedIDs...) }()
	for _, item := range changed {
		if updateErr := s.repo.UpdateIncident(ctx, item, item.Version); updateErr != nil {
			return ids, updateErr
		}
	}
	return ids, nil
}
