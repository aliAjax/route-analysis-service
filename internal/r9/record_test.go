package r9

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/route-analysis-service/internal/application"
	"github.com/example/route-analysis-service/internal/domain/model"
	"github.com/example/route-analysis-service/internal/infrastructure/memory"
	"github.com/example/route-analysis-service/internal/transport/httpapi"
)

func TestIncidentConflictsKeepOwnEdgeLists(t *testing.T) {
	store := memory.NewStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := application.NewService(store, logger)
	svc.SeedDemoData()
	ctx := context.Background()
	now := time.Now().UTC()
	window := model.TimeWindow{Start: now.Add(-time.Hour), End: now.Add(time.Hour)}
	incs := []model.Incident{
		{ID: "inc-a", DatasetID: "demo-city", EdgeIDs: []model.EdgeID{"ab"}, Status: model.IncidentActive, Window: window, SpeedFactor: 1, Priority: 5, CreatedAt: now},
		{ID: "inc-b", DatasetID: "demo-city", EdgeIDs: []model.EdgeID{"ab"}, Status: model.IncidentActive, Window: window, SpeedFactor: 1, Priority: 4, CreatedAt: now},
		{ID: "inc-c", DatasetID: "demo-city", EdgeIDs: []model.EdgeID{"bc"}, Status: model.IncidentActive, Window: window, SpeedFactor: 1, Priority: 3, CreatedAt: now},
		{ID: "inc-d", DatasetID: "demo-city", EdgeIDs: []model.EdgeID{"bc"}, Status: model.IncidentActive, Window: window, SpeedFactor: 1, Priority: 2, CreatedAt: now},
	}
	for _, inc := range incs {
		if _, err := svc.CreateIncident(ctx, inc); err != nil {
			t.Fatal(err)
		}
	}
	handler := httpapi.NewHandler(svc, logger).Routes()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents/conflicts?datasetId=demo-city&at="+now.Format(time.RFC3339), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var conflicts []struct {
		EdgeIDs []model.EdgeID `json:"edgeIds"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &conflicts); err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 2 {
		t.Fatalf("expected 2 conflicts, got %d", len(conflicts))
	}
	if len(conflicts[0].EdgeIDs) != 1 || conflicts[0].EdgeIDs[0] != "ab" {
		t.Fatalf("first conflict edges corrupted: %v", conflicts[0].EdgeIDs)
	}
	if len(conflicts[1].EdgeIDs) != 1 || conflicts[1].EdgeIDs[0] != "bc" {
		t.Fatalf("second conflict edges corrupted: %v", conflicts[1].EdgeIDs)
	}
}
