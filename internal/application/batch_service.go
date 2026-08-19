package application

import (
	"context"

	"github.com/example/route-analysis-service/internal/domain/analytics"
	"github.com/example/route-analysis-service/internal/domain/model"
	"github.com/example/route-analysis-service/internal/infrastructure/worker"
)

// ValidateDatasetsBatch validates several datasets concurrently and returns a
// report per dataset id. Cancellation stops scheduling remaining datasets.
func (s *Service) ValidateDatasetsBatch(ctx context.Context, ids []model.DatasetID) (map[model.DatasetID]analytics.Report, error) {
	if len(ids) == 0 {
		return map[model.DatasetID]analytics.Report{}, nil
	}
	results := make(map[model.DatasetID]analytics.Report, len(ids))
	dispatcher := worker.NewDispatcher(4)
	_, err := dispatcher.Run(ctx, datasetIDsToStrings(ids), func(ctx context.Context, raw string) error {
		report, validateErr := s.ValidateDataset(ctx, model.DatasetID(raw))
		results[model.DatasetID(raw)] = report
		return validateErr
	})
	if err != nil {
		return results, err
	}
	return results, nil
}

func datasetIDsToStrings(ids []model.DatasetID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, string(id))
	}
	return out
}
