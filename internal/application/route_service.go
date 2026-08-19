package application

import (
	"context"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// CachedRoute resolves a route through the in-memory LRU cache keyed by the
// normalised request. Cache misses fall through to the repository snapshot.
func (s *Service) CachedRoute(ctx context.Context, req model.RouteRequest) (model.RouteResult, error) {
	key := routeCacheKey(req)
	if value, ok := s.routeCache.Get(key); ok {
		if result, ok := value.(model.RouteResult); ok {
			return cloneRouteResult(result), nil
		}
	}
	result, err := s.Route(ctx, req)
	if err != nil {
		return model.RouteResult{}, err
	}
	copied := cloneRouteResult(result)
	s.routeCache.Put(key, copied)
	return cloneRouteResult(copied), nil
}

func cloneRouteResult(in model.RouteResult) model.RouteResult {
	out := in
	out.Legs = append([]model.RouteLeg(nil), in.Legs...)
	out.Explanation = append([]string(nil), in.Explanation...)
	for i := range out.Legs {
		out.Legs[i].IncidentIDs = append([]model.IncidentID(nil), in.Legs[i].IncidentIDs...)
	}
	return out
}

func routeCacheKey(req model.RouteRequest) string {
	return string(req.DatasetID) + "|" + string(req.From) + "|" + string(req.To) + "|" + string(req.Mode) + "|" + req.Departure.UTC().Format("2006-01-02T15:04:05Z07:00") + "|" + req.VehicleClass
}
