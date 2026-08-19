package r5

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/example/route-analysis-service/internal/application"
	"github.com/example/route-analysis-service/internal/domain/incident"
	"github.com/example/route-analysis-service/internal/domain/model"
	"github.com/example/route-analysis-service/internal/infrastructure/memory"
)

func TestResolveExpiredOnlyResolvesExpiredIncidents(t *testing.T) {
	store := memory.NewStore()
	svc := application.NewService(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.SeedDemoData()
	ctx := context.Background()
	now := time.Now().UTC()
	expired := model.Incident{ID: "inc-expired", DatasetID: "demo-city", EdgeIDs: []model.EdgeID{"ab"}, Status: model.IncidentActive, Window: model.TimeWindow{Start: now.Add(-2 * time.Hour), End: now.Add(-time.Hour)}, SpeedFactor: 1, Priority: 5, CreatedAt: now.Add(-3 * time.Hour)}
	future := model.Incident{ID: "inc-future", DatasetID: "demo-city", EdgeIDs: []model.EdgeID{"bc"}, Status: model.IncidentActive, Window: model.TimeWindow{Start: now, End: now.Add(2 * time.Hour)}, SpeedFactor: 1, Priority: 5, CreatedAt: now}
	if _, err := svc.CreateIncident(ctx, expired); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateIncident(ctx, future); err != nil {
		t.Fatal(err)
	}
	ids, err := svc.ResolveExpiredIncidents(ctx, "demo-city", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "inc-expired" {
		t.Fatalf("expected only expired id resolved, got %v", ids)
	}
	list, err := svc.ListIncidents(ctx, "demo-city")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range list {
		if item.ID == "inc-future" && item.Status != model.IncidentActive {
			t.Fatalf("future incident should stay active, got %s", item.Status)
		}
	}
}

func TestBatchTransitionReportsOnlySuccessfulItems(t *testing.T) {
	items := []model.Incident{
		{ID: "a", Status: model.IncidentDraft},
		{ID: "b", Status: model.IncidentRevoked},
	}
	updated, result := incident.BatchTransition(items, nil, model.IncidentActive)
	if len(updated) != 2 {
		t.Fatalf("expected 2 updated items, got %d", len(updated))
	}
	if len(result.Updated) != 1 || result.Updated[0] != "a" {
		t.Fatalf("expected only 'a' as successfully updated, got %v", result.Updated)
	}
	if len(result.Failed) != 1 || result.Failed[0].Incident != "b" {
		t.Fatalf("expected 'b' to fail, got %v", result.Failed)
	}
}
