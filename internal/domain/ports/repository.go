package ports

import (
	"context"
	"github.com/example/route-analysis-service/internal/domain/model"
)

type Repository interface {
	CreateDataset(context.Context, model.Dataset) error
	GetDataset(context.Context, model.DatasetID) (model.Dataset, error)
	ListDatasets(context.Context) ([]model.Dataset, error)
	PutGraph(context.Context, model.DatasetID, []model.Node, []model.Edge) error
	Graph(context.Context, model.DatasetID) ([]model.Node, []model.Edge, int64, error)
	CreateIncident(context.Context, model.Incident) error
	UpdateIncident(context.Context, model.Incident, int64) error
	ListIncidents(context.Context, model.DatasetID) ([]model.Incident, error)
	GetIncident(context.Context, model.DatasetID, model.IncidentID) (model.Incident, error)
	CreateJob(context.Context, model.AnalysisJob) error
	UpdateJob(context.Context, model.AnalysisJob) error
	GetJob(context.Context, model.JobID) (model.AnalysisJob, error)
	ListJobs(context.Context, model.DatasetID) ([]model.AnalysisJob, error)
}
