package r4

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/example/route-analysis-service/internal/application"
	"github.com/example/route-analysis-service/internal/domain/model"
	"github.com/example/route-analysis-service/internal/infrastructure/memory"
)

type countingContext struct {
	context.Context
	calls int
}

func (c *countingContext) Err() error {
	c.calls++
	if c.calls >= 2 {
		return context.Canceled
	}
	return nil
}

func TestNearestNodeRespectsCancellation(t *testing.T) {
	store := memory.NewStore()
	svc := application.NewService(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.SeedDemoData()
	ctx := &countingContext{Context: context.Background()}
	_, _, err := svc.NearestNode(ctx, "demo-city", model.Point{Lat: 35.68, Lon: 139.76}, model.ModeDrive)
	if err == nil {
		t.Fatal("expected cancellation error from NearestNode")
	}
}
