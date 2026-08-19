package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/example/route-analysis-service/internal/domain/incident"
	"github.com/example/route-analysis-service/internal/domain/model"
	"github.com/example/route-analysis-service/internal/domain/ports"
	"github.com/example/route-analysis-service/internal/infrastructure/memory"
	"github.com/example/route-analysis-service/internal/domain/quota"
	"github.com/example/route-analysis-service/internal/domain/routing"
	"github.com/example/route-analysis-service/internal/infrastructure/cache"
	"log/slog"
	"math"
	"sync"
	"time"
)

type Service struct {
	repo       ports.Repository
	logger     *slog.Logger
	jobs       chan model.JobID
	stop       chan struct{}
	once       sync.Once
	wg         sync.WaitGroup
	quota      *quota.Evaluator
	routeCache *cache.LRU
	resolver   *incident.Resolver
}

func NewService(repo ports.Repository, logger *slog.Logger) *Service {
	s := &Service{
		repo:       repo,
		logger:     logger,
		jobs:       make(chan model.JobID, 64),
		stop:       make(chan struct{}),
		quota:      quota.NewEvaluator(nil),
		routeCache: cache.NewLRU(512),
		resolver:   incident.NewResolver(nil),
	}
	s.wg.Add(1)
	go s.worker()
	return s
}
func (s *Service) Close() { s.once.Do(func() { close(s.stop); s.wg.Wait() }) }
func (s *Service) CreateDataset(ctx context.Context, name, description string, tags map[string]string) (model.Dataset, error) {
	now := time.Now().UTC()
	d := model.Dataset{ID: model.DatasetID(newID("ds")), Name: name, Description: description, Tags: tags, Version: 1, Status: model.DatasetDraft, CreatedAt: now, UpdatedAt: now}
	return d, s.repo.CreateDataset(ctx, d)
}
func (s *Service) ListDatasets(ctx context.Context) ([]model.Dataset, error) {
	return s.repo.ListDatasets(ctx)
}
func (s *Service) ImportGraph(ctx context.Context, id model.DatasetID, nodes []model.Node, edges []model.Edge) error {
	return s.repo.PutGraph(ctx, id, nodes, edges)
}
func (s *Service) snapshot(ctx context.Context, id model.DatasetID) (routing.Snapshot, error) {
	nodes, edges, version, err := s.repo.Graph(ctx, id)
	if err != nil {
		return routing.Snapshot{}, err
	}
	incidents, err := s.repo.ListIncidents(ctx, id)
	if err != nil {
		return routing.Snapshot{}, err
	}
	return routing.NewSnapshot(nodes, edges, incidents, version), nil
}
func (s *Service) Route(ctx context.Context, req model.RouteRequest) (model.RouteResult, error) {
	snapshot, err := s.snapshot(ctx, req.DatasetID)
	if err != nil {
		return model.RouteResult{}, err
	}
	return snapshot.Route(req)
}
func (s *Service) KRoutes(ctx context.Context, req model.RouteRequest, k int) ([]model.RouteResult, error) {
	snapshot, err := s.snapshot(ctx, req.DatasetID)
	if err != nil {
		return nil, err
	}
	return snapshot.KRoutes(req, k)
}
func (s *Service) Isochrone(ctx context.Context, req model.RouteRequest, budget float64) (map[model.NodeID]float64, error) {
	snapshot, err := s.snapshot(ctx, req.DatasetID)
	if err != nil {
		return nil, err
	}
	return snapshot.Reachable(req, budget)
}
func (s *Service) CreateIncident(ctx context.Context, incident model.Incident) (model.Incident, error) {
	if incident.ID == "" {
		incident.ID = model.IncidentID(newID("inc"))
	}
	if incident.CreatedAt.IsZero() {
		incident.CreatedAt = time.Now().UTC()
	}
	if err := s.repo.CreateIncident(ctx, incident); err != nil {
		return model.Incident{}, err
	}
	return incident, nil
}
func (s *Service) ActivateIncident(ctx context.Context, id model.IncidentID, dataset model.DatasetID, expected int64) error {
	incidents, err := s.repo.ListIncidents(ctx, dataset)
	if err != nil {
		return err
	}
	for _, item := range incidents {
		if item.ID == id {
			item.Status = model.IncidentActive
			if updateErr := s.repo.UpdateIncident(ctx, item, expected); updateErr != nil {
				return fmt.Errorf("activate incident %s: %w", id, updateErr)
			}
			return nil
		}
	}
	return fmt.Errorf("incident %s: %w", id, memory.ErrNotFound)
}
func (s *Service) ListIncidents(ctx context.Context, id model.DatasetID) ([]model.Incident, error) {
	return s.repo.ListIncidents(ctx, id)
}
func (s *Service) CreateJob(ctx context.Context, kind model.JobType, datasetID model.DatasetID, parameters map[string]any) (model.AnalysisJob, error) {
	job := model.AnalysisJob{ID: model.JobID(newID("job")), Type: kind, DatasetID: datasetID, Status: model.JobQueued, Parameters: parameters, CreatedAt: time.Now().UTC()}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		return model.AnalysisJob{}, err
	}
	select {
	case s.jobs <- job.ID:
	default:
		return job, errors.New("analysis queue full")
	}
	return job, nil
}
func (s *Service) GetJob(ctx context.Context, id model.JobID) (model.AnalysisJob, error) {
	return s.repo.GetJob(ctx, id)
}
func (s *Service) ListJobs(ctx context.Context, id model.DatasetID) ([]model.AnalysisJob, error) {
	return s.repo.ListJobs(ctx, id)
}
func (s *Service) worker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stop:
			return
		case id := <-s.jobs:
			s.runJob(id)
		}
	}
}
func (s *Service) runJob(id model.JobID) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	job, err := s.repo.GetJob(ctx, id)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	job.Status = model.JobRunning
	job.StartedAt = &now
	job.Progress = 5
	_ = s.repo.UpdateJob(ctx, job)
	snapshot, err := s.snapshot(ctx, job.DatasetID)
	if err == nil {
		job.Progress = 35
		_ = s.repo.UpdateJob(ctx, job)
		job.Result, err = s.analyze(ctx, snapshot, job)
		job.Progress = 90
		_ = s.repo.UpdateJob(ctx, job)
	}
	done := time.Now().UTC()
	job.CompletedAt = &done
	if err != nil {
		job.Status = model.JobFailed
		job.Error = err.Error()
	} else {
		job.Status = model.JobSucceeded
		job.Progress = 100
	}
	_ = s.repo.UpdateJob(ctx, job)
	s.logger.Info("analysis job completed", "job", id, "status", job.Status)
}
func (s *Service) analyze(ctx context.Context, snapshot routing.Snapshot, job model.AnalysisJob) (map[string]any, error) {
	switch job.Type {
	case model.JobCentrality:
		values := snapshot.DegreeCentrality()
		return map[string]any{"algorithm": "normalized_degree", "values": values, "nodeCount": len(values)}, nil
	case model.JobIsochrone:
		from, ok := job.Parameters["from"].(string)
		if !ok {
			return nil, errors.New("from parameter required")
		}
		budget := number(job.Parameters["budgetSeconds"], 600)
		reachable, err := snapshot.Reachable(model.RouteRequest{DatasetID: job.DatasetID, From: model.NodeID(from), To: "placeholder", Mode: model.ModeDrive, Departure: time.Now().UTC()}, budget)
		if err != nil && len(reachable) == 0 {
			return nil, err
		}
		return map[string]any{"budgetSeconds": budget, "reachable": reachable, "count": len(reachable)}, nil
	case model.JobODMatrix:
		origins := strings(job.Parameters["origins"])
		destinations := strings(job.Parameters["destinations"])
		if len(origins) == 0 || len(destinations) == 0 {
			return nil, errors.New("origins and destinations required")
		}
		matrix := map[string]float64{}
		for _, from := range origins {
			for _, to := range destinations {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				if from == to {
					matrix[from+"->"+to] = 0
					continue
				}
				result, routeErr := snapshot.Route(model.RouteRequest{DatasetID: job.DatasetID, From: model.NodeID(from), To: model.NodeID(to), Mode: model.ModeDrive, Departure: time.Now().UTC()})
				if routeErr == nil {
					matrix[from+"->"+to] = math.Round(result.TotalSeconds*100) / 100
				}
			}
		}
		return map[string]any{"matrix": matrix, "origins": origins, "destinations": destinations}, nil
	default:
		return nil, fmt.Errorf("unsupported job type %s", job.Type)
	}
}
func number(value any, fallback float64) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return fallback
	}
}
func strings(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}
func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
func (s *Service) SeedDemoData() {
	ctx := context.Background()
	d := model.Dataset{ID: "demo-city", Name: "Demo City Network", Version: 1, Status: model.DatasetReady, Description: "A compact directed network for smoke testing", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), Tags: map[string]string{"source": "seed"}}
	if err := s.repo.CreateDataset(ctx, d); err != nil {
		return
	}
	points := map[model.NodeID]model.Point{"A": {Lat: 35.6812, Lon: 139.7671}, "B": {Lat: 35.6820, Lon: 139.7750}, "C": {Lat: 35.6762, Lon: 139.7795}, "D": {Lat: 35.6721, Lon: 139.7690}, "E": {Lat: 35.6760, Lon: 139.7600}, "F": {Lat: 35.6845, Lon: 139.7580}}
	nodes := make([]model.Node, 0, len(points))
	for id, point := range points {
		nodes = append(nodes, model.Node{ID: id, DatasetID: d.ID, Location: point, Kind: "intersection", Name: string(id)})
	}
	makeEdge := func(id model.EdgeID, from, to model.NodeID, seconds float64, toll int) model.Edge {
		return model.Edge{ID: id, DatasetID: d.ID, From: from, To: to, LengthMeters: seconds * 12, BaseSeconds: seconds, Capacity: 1000, TollCents: toll, Modes: []model.Mode{model.ModeDrive, model.ModeBike, model.ModeWalk}}
	}
	edges := []model.Edge{makeEdge("ab", "A", "B", 40, 0), makeEdge("bc", "B", "C", 40, 100), makeEdge("cd", "C", "D", 45, 0), makeEdge("de", "D", "E", 35, 0), makeEdge("ef", "E", "F", 30, 0), makeEdge("fa", "F", "A", 35, 0), makeEdge("be", "B", "E", 95, 0), makeEdge("ad", "A", "D", 120, 0), makeEdge("ce", "C", "E", 45, 0), makeEdge("ba", "B", "A", 45, 0), makeEdge("cb", "C", "B", 40, 100), makeEdge("dc", "D", "C", 45, 0), makeEdge("ed", "E", "D", 35, 0), makeEdge("fe", "F", "E", 30, 0), makeEdge("af", "A", "F", 35, 0)}
	_ = s.repo.PutGraph(ctx, d.ID, nodes, edges)
}
