package r3

import (
	"context"
	"io"
	"log/slog"
	strpkg "strings"
	"testing"

	"github.com/example/route-analysis-service/internal/application"
	"github.com/example/route-analysis-service/internal/infrastructure/memory"
)

func newReportService() *application.Service {
	store := memory.NewStore()
	svc := application.NewService(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc.SeedDemoData()
	return svc
}

func TestBuildNetworkReportCSVAndSummariseIndependent(t *testing.T) {
	svc := newReportService()
	ctx := context.Background()
	report, err := svc.BuildNetworkReport(ctx, "demo-city")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Sections) != 2 || report.Sections[0].Name != "network" || len(report.Sections[0].Rows) != 4 {
		t.Fatalf("network section corrupted: %v", report.Sections)
	}
	if report.Sections[0].Rows[0][0] != "metric" || report.Sections[0].Rows[1][0] != "nodes" {
		t.Fatalf("network rows overwritten: %v", report.Sections[0].Rows)
	}
	csvBytes, err := report.ToCSV()
	if err != nil {
		t.Fatal(err)
	}
	csvText := string(csvBytes)
	if !strpkg.Contains(csvText, "[section:network]") || !strpkg.Contains(csvText, "metric") || !strpkg.Contains(csvText, "toll_edges") {
		t.Fatalf("csv lost rows: %s", csvText)
	}
	if report.Sections[0].Rows[0][0] != "metric" {
		t.Fatalf("ToCSV mutated report: %v", report.Sections[0].Rows)
	}
	first, err := svc.SummariseDataset(ctx, "demo-city")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.SummariseDataset(ctx, "demo-city")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Modes) != len(second.Modes) || len(first.Modes) == 0 {
		t.Fatal("mode lists invalid")
	}
	first.Modes[0] = "mutated-mode"
	if second.Modes[0] == "mutated-mode" {
		t.Fatal("summaries share mutable backing array")
	}
}

func TestBuildIncidentReportRowsIndependent(t *testing.T) {
	svc := newReportService()
	report, err := svc.BuildIncidentReport(context.Background(), "demo-city")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Sections) != 2 || report.Sections[0].Name != "incidents" || len(report.Sections[0].Rows) != 1 {
		t.Fatalf("incidents section corrupted: %v", report.Sections)
	}
	if report.Sections[0].Rows[0][0] != "incident" {
		t.Fatalf("incidents rows overwritten: %v", report.Sections[0].Rows)
	}
}
