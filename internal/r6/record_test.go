package r6

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/example/route-analysis-service/internal/application"
	"github.com/example/route-analysis-service/internal/domain/model"
	"github.com/example/route-analysis-service/internal/infrastructure/memory"
)

func TestCachedRouteConcurrentReadsNoRace(t *testing.T) {
	store := memory.NewStore()
	svc := application.NewService(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.SeedDemoData()
	req := model.RouteRequest{DatasetID: "demo-city", From: "A", To: "D", Mode: model.ModeDrive, Departure: time.Now().UTC()}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 100; j++ {
				result, err := svc.CachedRoute(context.Background(), req)
				if err != nil || result.TotalSeconds <= 0 {
					t.Errorf("bad cached route: err=%v seconds=%f", err, result.TotalSeconds)
					return
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}
