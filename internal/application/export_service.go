package application

import (
	"context"
	"fmt"
	"time"

	"github.com/example/route-analysis-service/internal/domain/export"
	"github.com/example/route-analysis-service/internal/domain/model"
)

// BuildNetworkReport produces a network quality report for a dataset.
func (s *Service) BuildNetworkReport(ctx context.Context, id model.DatasetID) (export.Report, error) {
	nodes, edges, _, err := s.repo.Graph(ctx, id)
	if err != nil {
		return export.Report{}, err
	}
	summary := export.SummariseNetwork(nodes, edges)
	report := export.NewReport(newID("rep"), "Network quality report", export.ReportNetworkQuality, id)
	report.AddSection(export.Section{Name: "network", Rows: [][]string{
		{"metric", "value"},
		{"nodes", fmt.Sprintf("%d", summary.Nodes)},
		{"edges", fmt.Sprintf("%d", summary.Edges)},
		{"toll_edges", fmt.Sprintf("%d", summary.TollEdges)},
	}, Totals: map[string]any{"max_slope": summary.MaxSlope}})
	report.AddSection(export.Section{Name: "modes", Rows: [][]string{{"mode"}, {"drive"}, {"bike"}, {"walk"}, {"transit"}}})
	return report, nil
}

// BuildIncidentReport produces an incident log report for a dataset.
func (s *Service) BuildIncidentReport(ctx context.Context, id model.DatasetID) (export.Report, error) {
	items, err := s.ListIncidentsSorted(ctx, id)
	if err != nil {
		return export.Report{}, err
	}
	report := export.NewReport(newID("rep"), "Incident log", export.ReportIncidentLog, id)
	rows := [][]string{{"incident", "status", "priority", "edges"}}
	for _, item := range items {
		rows = append(rows, []string{string(item.ID), string(item.Status), fmt.Sprintf("%d", item.Priority), fmt.Sprintf("%d", len(item.EdgeIDs))})
	}
	report.AddSection(export.Section{Name: "incidents", Rows: rows, Totals: map[string]any{"count": len(items)}})
	return report, nil
}

// ExportODMatrix computes a drive-time OD matrix and returns it as CSV bytes.
func (s *Service) ExportODMatrix(ctx context.Context, id model.DatasetID, origins, destinations []model.NodeID) ([]byte, error) {
	snapshot, err := s.snapshot(ctx, id)
	if err != nil {
		return nil, err
	}
	rows := [][]string{{"origin", "destination", "seconds"}}
	for _, from := range origins {
		for _, to := range destinations {
			if from == to {
				rows = append(rows, []string{string(from), string(to), "0"})
				continue
			}
			result, routeErr := snapshot.Route(model.RouteRequest{DatasetID: id, From: from, To: to, Mode: model.ModeDrive, Departure: time.Now().UTC()})
			if routeErr == nil {
				rows = append(rows, []string{string(from), string(to), fmt.Sprintf("%.2f", result.TotalSeconds)})
			}
		}
	}
	report := export.NewReport(newID("rep"), "OD matrix", export.ReportODSummary, id)
	report.AddSection(export.Section{Name: "matrix", Rows: rows, Totals: map[string]any{"origins": len(origins), "destinations": len(destinations)}})
	return report.ToCSV()
}
