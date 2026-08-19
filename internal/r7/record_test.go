package r7

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/example/route-analysis-service/internal/application"
	"github.com/example/route-analysis-service/internal/domain/model"
	"github.com/example/route-analysis-service/internal/infrastructure/memory"
)

func TestValidateDatasetsBatchConcurrentNoRace(t *testing.T) {
	store := memory.NewStore()
	svc := application.NewService(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.SeedDemoData()
	ctx := context.Background()
	ids := []model.DatasetID{"demo-city"}
	for _, name := range []string{"ds-a", "ds-b", "ds-c"} {
		ds, createErr := svc.CreateDataset(ctx, name, name, map[string]string{})
		if createErr != nil {
			t.Fatal(createErr)
		}
		if importErr := svc.ImportGraph(ctx, ds.ID, nil, nil); importErr != nil {
			t.Fatal(importErr)
		}
		ids = append(ids, ds.ID)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results, err := svc.ValidateDatasetsBatch(ctx, ids)
			if err != nil || len(results) != len(ids) {
				t.Errorf("batch validation failed: err=%v results=%d", err, len(results))
				return
			}
		}()
	}
	close(start)
	wg.Wait()
}
