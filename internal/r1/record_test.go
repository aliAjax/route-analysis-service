package r1

import (
	"context"
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

func newActivateEnv() (*application.Service, http.Handler) {
	store := memory.NewStore()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := application.NewService(store, logger)
	svc.SeedDemoData()
	return svc, httpapi.NewHandler(svc, logger).Routes()
}

func TestActivateMissingIncidentReturnsNotFound(t *testing.T) {
	_, handler := newActivateEnv()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/inc-missing/activate?datasetId=demo-city", nil)
	req.Header.Set("If-Match", "1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivateStaleVersionReturnsConflict(t *testing.T) {
	svc, handler := newActivateEnv()
	inc := model.Incident{ID: "inc-1", DatasetID: "demo-city", EdgeIDs: []model.EdgeID{"ab"}, Status: model.IncidentDraft, Window: model.TimeWindow{}, SpeedFactor: 1, Priority: 10, CreatedAt: time.Now().UTC()}
	if _, err := svc.CreateIncident(context.Background(), inc); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/inc-1/activate?datasetId=demo-city", nil)
	req.Header.Set("If-Match", "99")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}
