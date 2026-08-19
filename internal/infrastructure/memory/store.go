package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/example/route-analysis-service/internal/domain/model"
)

var ErrNotFound = errors.New("resource not found")
var ErrConflict = errors.New("resource conflict")

type graph struct {
	nodes   map[model.NodeID]model.Node
	edges   map[model.EdgeID]model.Edge
	version int64
}
type Store struct {
	mu        sync.RWMutex
	datasets  map[model.DatasetID]model.Dataset
	graphs    map[model.DatasetID]*graph
	incidents map[model.DatasetID]map[model.IncidentID]model.Incident
	jobs      map[model.JobID]model.AnalysisJob
}

func NewStore() *Store {
	return &Store{datasets: map[model.DatasetID]model.Dataset{}, graphs: map[model.DatasetID]*graph{}, incidents: map[model.DatasetID]map[model.IncidentID]model.Incident{}, jobs: map[model.JobID]model.AnalysisJob{}}
}
func (s *Store) CreateDataset(_ context.Context, d model.Dataset) error {
	if err := d.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.datasets[d.ID]; ok {
		return ErrConflict
	}
	s.datasets[d.ID] = cloneDataset(d)
	return nil
}
func (s *Store) GetDataset(_ context.Context, id model.DatasetID) (model.Dataset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.datasets[id]
	if !ok {
		return model.Dataset{}, ErrNotFound
	}
	return cloneDataset(d), nil
}
func (s *Store) ListDatasets(_ context.Context) ([]model.Dataset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Dataset, 0, len(s.datasets))
	for _, d := range s.datasets {
		out = append(out, cloneDataset(d))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
func (s *Store) PutGraph(_ context.Context, id model.DatasetID, nodes []model.Node, edges []model.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.datasets[id]; !ok {
		return ErrNotFound
	}
	g := &graph{nodes: map[model.NodeID]model.Node{}, edges: map[model.EdgeID]model.Edge{}, version: 1}
	if old := s.graphs[id]; old != nil {
		g.version = old.version + 1
	}
	for _, n := range nodes {
		if err := n.Validate(); err != nil {
			return err
		}
		if n.DatasetID != id {
			return errors.New("node dataset mismatch")
		}
		if _, ok := g.nodes[n.ID]; ok {
			return ErrConflict
		}
		g.nodes[n.ID] = cloneNode(n)
	}
	for _, e := range edges {
		if err := e.Validate(); err != nil {
			return err
		}
		if e.DatasetID != id {
			return errors.New("edge dataset mismatch")
		}
		if _, ok := g.nodes[e.From]; !ok {
			return errors.New("edge origin missing")
		}
		if _, ok := g.nodes[e.To]; !ok {
			return errors.New("edge destination missing")
		}
		if _, ok := g.edges[e.ID]; ok {
			return ErrConflict
		}
		g.edges[e.ID] = cloneEdge(e)
	}
	s.graphs[id] = g
	d := s.datasets[id]
	d.Version = g.version
	s.datasets[id] = d
	return nil
}
func (s *Store) Graph(_ context.Context, id model.DatasetID) ([]model.Node, []model.Edge, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g := s.graphs[id]
	if g == nil {
		return nil, nil, 0, ErrNotFound
	}
	nodes := make([]model.Node, 0, len(g.nodes))
	edges := make([]model.Edge, 0, len(g.edges))
	for _, n := range g.nodes {
		nodes = append(nodes, cloneNode(n))
	}
	for _, e := range g.edges {
		edges = append(edges, cloneEdge(e))
	}
	return nodes, edges, g.version, nil
}
func (s *Store) CreateIncident(_ context.Context, i model.Incident) error {
	if err := i.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.datasets[i.DatasetID]; !ok {
		return ErrNotFound
	}
	if s.incidents[i.DatasetID] == nil {
		s.incidents[i.DatasetID] = map[model.IncidentID]model.Incident{}
	}
	if _, ok := s.incidents[i.DatasetID][i.ID]; ok {
		return fmt.Errorf("incident %s: %w", i.ID, ErrConflict)
	}
	i.Version = 1
	s.incidents[i.DatasetID][i.ID] = cloneIncident(i)
	return nil
}
func (s *Store) UpdateIncident(_ context.Context, i model.Incident, expected int64) error {
	if err := i.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.incidents[i.DatasetID][i.ID]
	if !ok {
		return fmt.Errorf("incident %s: %w", i.ID, ErrNotFound)
	}
	if current.Version != expected {
		return fmt.Errorf("incident %s: %w", i.ID, ErrConflict)
	}
	i.Version = current.Version + 1
	s.incidents[i.DatasetID][i.ID] = cloneIncident(i)
	return nil
}
func (s *Store) ListIncidents(_ context.Context, id model.DatasetID) ([]model.Incident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Incident, 0, len(s.incidents[id]))
	for _, i := range s.incidents[id] {
		out = append(out, cloneIncident(i))
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Priority > out[b].Priority })
	return out, nil
}
func (s *Store) GetIncident(_ context.Context, dataset model.DatasetID, id model.IncidentID) (model.Incident, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	incidents := s.incidents[dataset]
	item, ok := incidents[id]
	if !ok {
		return model.Incident{}, fmt.Errorf("incident %s: %w", id, ErrNotFound)
	}
	return cloneIncident(item), nil
}

func (s *Store) CreateJob(_ context.Context, j model.AnalysisJob) error {
	if err := j.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.datasets[j.DatasetID]; !ok {
		return ErrNotFound
	}
	if _, ok := s.jobs[j.ID]; ok {
		return ErrConflict
	}
	s.jobs[j.ID] = cloneJob(j)
	return nil
}
func (s *Store) UpdateJob(_ context.Context, j model.AnalysisJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[j.ID]; !ok {
		return ErrNotFound
	}
	s.jobs[j.ID] = cloneJob(j)
	return nil
}
func (s *Store) GetJob(_ context.Context, id model.JobID) (model.AnalysisJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return model.AnalysisJob{}, ErrNotFound
	}
	return cloneJob(j), nil
}
func (s *Store) ListJobs(_ context.Context, id model.DatasetID) ([]model.AnalysisJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []model.AnalysisJob{}
	for _, j := range s.jobs {
		if j.DatasetID == id {
			out = append(out, cloneJob(j))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
func cloneDataset(v model.Dataset) model.Dataset { v.Tags = cloneStringMap(v.Tags); return v }
func cloneNode(v model.Node) model.Node          { v.Metadata = cloneStringMap(v.Metadata); return v }
func cloneEdge(v model.Edge) model.Edge {
	v.Modes = append([]model.Mode(nil), v.Modes...)
	v.Rules = append([]model.AccessRule(nil), v.Rules...)
	v.Geometry = append([]model.Point(nil), v.Geometry...)
	v.Metadata = cloneStringMap(v.Metadata)
	return v
}
func cloneIncident(v model.Incident) model.Incident {
	v.EdgeIDs = append([]model.EdgeID(nil), v.EdgeIDs...)
	return v
}
func cloneJob(v model.AnalysisJob) model.AnalysisJob {
	v.Parameters = cloneAnyMap(v.Parameters)
	v.Result = cloneAnyMap(v.Result)
	return v
}
func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
