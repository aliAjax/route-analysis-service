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
			result.Explanation = tagExplanation(result.Explanation, "cache-hit")
			return result, nil
		}
	}
	result, err := s.Route(ctx, req)
	if err != nil {
		return model.RouteResult{}, err
	}
	s.routeCache.Put(key, result)
	result.Explanation = tagExplanation(result.Explanation, "cache-miss")
	return result, nil
}

// tagExplanation returns a copy of explanation with its first element
// replaced by tag. Copying avoids mutating the backing array that the cached
// value still references, which would race with concurrent cache hits.
func tagExplanation(explanation []string, tag string) []string {
	if len(explanation) == 0 {
		return explanation
	}
	tagged := make([]string, len(explanation))
	copy(tagged, explanation)
	tagged[0] = tag
	return tagged
}

func routeCacheKey(req model.RouteRequest) string {
	return string(req.DatasetID) + "|" + string(req.From) + "|" + string(req.To) + "|" + string(req.Mode) + "|" + req.Departure.UTC().Format("2006-01-02T15:04:05Z07:00") + "|" + req.VehicleClass
}
