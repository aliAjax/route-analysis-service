package export

import (
	"errors"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// ReportKind enumerates the report types this service can emit.
type ReportKind string

const (
	ReportNetworkQuality ReportKind = "network_quality"
	ReportODSummary      ReportKind = "od_summary"
	ReportIncidentLog    ReportKind = "incident_log"
)

func (k ReportKind) Valid() bool {
	switch k {
	case ReportNetworkQuality, ReportODSummary, ReportIncidentLog:
		return true
	default:
		return false
	}
}

// Report is the domain model for a generated report.
type Report struct {
	ID        string          `json:"id"`
	Kind      ReportKind      `json:"kind"`
	DatasetID model.DatasetID `json:"datasetId"`
	Title     string          `json:"title"`
	Generated time.Time       `json:"generated"`
	Sections  []Section       `json:"sections"`
}

// Section is one named block inside a report.
type Section struct {
	Name   string         `json:"name"`
	Rows   [][]string     `json:"rows"`
	Totals map[string]any `json:"totals,omitempty"`
}

func NewReport(id, title string, kind ReportKind, dataset model.DatasetID) Report {
	return Report{ID: id, Kind: kind, DatasetID: dataset, Title: title, Generated: time.Now().UTC(), Sections: []Section{}}
}

// AddSection appends a report section.
func (r *Report) AddSection(section Section) {
	r.Sections = append(r.Sections, section)
}

// Validate performs a light integrity check on the report.
func (r Report) Validate() error {
	if r.ID == "" || r.DatasetID == "" {
		return errors.New("report id and dataset are required")
	}
	if !r.Kind.Valid() {
		return errors.New("unsupported report kind")
	}
	return nil
}
