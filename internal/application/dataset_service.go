package application

import (
	"context"

	"github.com/example/route-analysis-service/internal/domain/analytics"
	"github.com/example/route-analysis-service/internal/domain/export"
	"github.com/example/route-analysis-service/internal/domain/geocode"
	"github.com/example/route-analysis-service/internal/domain/model"
)

func (s *Service) GetDataset(ctx context.Context, id model.DatasetID) (model.Dataset, error) {
	return s.repo.GetDataset(ctx, id)
}

// ValidateDataset runs the graph validator over a dataset snapshot and returns
// a machine-readable report suitable for API consumers.
func (s *Service) ValidateDataset(ctx context.Context, id model.DatasetID) (analytics.Report, error) {
	nodes, edges, _, err := s.repo.Graph(ctx, id)
	if err != nil {
		return analytics.Report{}, err
	}
	report := analytics.ValidateGraph(id, nodes, edges)
	return report, nil
}

// SummariseDataset returns a compact network summary for a dataset.
func (s *Service) SummariseDataset(ctx context.Context, id model.DatasetID) (export.NetworkSummary, error) {
	nodes, edges, _, err := s.repo.Graph(ctx, id)
	if err != nil {
		return export.NetworkSummary{}, err
	}
	return export.SummariseNetwork(nodes, edges), nil
}

// NearestNode finds the closest routable node for the given point and mode.
func (s *Service) NearestNode(ctx context.Context, id model.DatasetID, point model.Point, mode model.Mode) (model.NodeID, float64, error) {
	snapshot, err := s.snapshot(ctx, id)
	if err != nil {
		return "", 0, err
	}
	return snapshot.Nearest(point, mode)
}

// ClusterNodes groups dataset nodes into geographic clusters.
func (s *Service) ClusterNodes(ctx context.Context, id model.DatasetID, radiusMeters float64) ([]geocode.Cluster, error) {
	nodes, _, _, err := s.repo.Graph(ctx, id)
	if err != nil {
		return nil, err
	}
	points := make(map[model.NodeID]model.Point, len(nodes))
	for _, node := range nodes {
		points[node.ID] = node.Location
	}
	return geocode.ClusterNodes(points, radiusMeters), nil
}
