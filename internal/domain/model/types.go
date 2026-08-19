package model

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type DatasetID string
type NodeID string
type EdgeID string
type JobID string
type IncidentID string

type Mode string

const (
	ModeDrive   Mode = "drive"
	ModeWalk    Mode = "walk"
	ModeBike    Mode = "bike"
	ModeTransit Mode = "transit"
)

func (m Mode) Valid() bool {
	return m == ModeDrive || m == ModeWalk || m == ModeBike || m == ModeTransit
}

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func (p Point) Valid() bool { return p.Lat >= -90 && p.Lat <= 90 && p.Lon >= -180 && p.Lon <= 180 }
func (p Point) DistanceMeters(to Point) float64 {
	const radius = 6371000.0
	lat1, lat2 := p.Lat*math.Pi/180, to.Lat*math.Pi/180
	dLat, dLon := lat2-lat1, (to.Lon-p.Lon)*math.Pi/180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * radius * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func (w TimeWindow) Valid() bool { return w.Start.IsZero() || w.End.IsZero() || !w.End.Before(w.Start) }
func (w TimeWindow) Contains(at time.Time) bool {
	return (w.Start.IsZero() || !at.Before(w.Start)) && (w.End.IsZero() || !at.After(w.End))
}

type DatasetStatus string

const (
	DatasetDraft     DatasetStatus = "draft"
	DatasetImporting DatasetStatus = "importing"
	DatasetReady     DatasetStatus = "ready"
	DatasetFailed    DatasetStatus = "failed"
	DatasetArchived  DatasetStatus = "archived"
)

type Dataset struct {
	ID          DatasetID         `json:"id"`
	Name        string            `json:"name"`
	Version     int64             `json:"version"`
	Status      DatasetStatus     `json:"status"`
	Description string            `json:"description"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	Tags        map[string]string `json:"tags"`
}

func (d Dataset) Validate() error {
	if strings.TrimSpace(d.Name) == "" {
		return errors.New("dataset name is required")
	}
	if d.Version < 1 {
		return errors.New("dataset version must be positive")
	}
	return nil
}

type Node struct {
	ID        NodeID            `json:"id"`
	DatasetID DatasetID         `json:"datasetId"`
	Location  Point             `json:"location"`
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Metadata  map[string]string `json:"metadata"`
}

func (n Node) Validate() error {
	if n.ID == "" || n.DatasetID == "" {
		return errors.New("node id and dataset id are required")
	}
	if !n.Location.Valid() {
		return errors.New("invalid node location")
	}
	return nil
}

type AccessRule struct {
	Mode         Mode       `json:"mode"`
	Window       TimeWindow `json:"window"`
	Allowed      bool       `json:"allowed"`
	VehicleClass string     `json:"vehicleClass"`
}

func (a AccessRule) Applies(mode Mode, at time.Time, vehicle string) bool {
	return a.Mode == mode && a.Window.Contains(at) && (a.VehicleClass == "" || a.VehicleClass == vehicle)
}

type Edge struct {
	ID           EdgeID            `json:"id"`
	DatasetID    DatasetID         `json:"datasetId"`
	From         NodeID            `json:"from"`
	To           NodeID            `json:"to"`
	LengthMeters float64           `json:"lengthMeters"`
	BaseSeconds  float64           `json:"baseSeconds"`
	Capacity     int               `json:"capacity"`
	TollCents    int               `json:"tollCents"`
	SlopePercent float64           `json:"slopePercent"`
	Modes        []Mode            `json:"modes"`
	Rules        []AccessRule      `json:"rules"`
	Geometry     []Point           `json:"geometry"`
	Metadata     map[string]string `json:"metadata"`
}

func (e Edge) Validate() error {
	if e.ID == "" || e.DatasetID == "" || e.From == "" || e.To == "" {
		return errors.New("edge identifiers are required")
	}
	if e.From == e.To {
		return errors.New("self edges are not allowed")
	}
	if e.LengthMeters <= 0 || e.BaseSeconds <= 0 {
		return errors.New("edge length and duration must be positive")
	}
	if len(e.Modes) == 0 {
		return errors.New("edge must support a mode")
	}
	for _, m := range e.Modes {
		if !m.Valid() {
			return fmt.Errorf("unsupported mode %q", m)
		}
	}
	return nil
}
func (e Edge) Supports(mode Mode, at time.Time, vehicle string) bool {
	found := false
	for _, m := range e.Modes {
		if m == mode {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	for _, rule := range e.Rules {
		if rule.Applies(mode, at, vehicle) {
			return rule.Allowed
		}
	}
	return true
}

type IncidentStatus string

const (
	IncidentDraft    IncidentStatus = "draft"
	IncidentActive   IncidentStatus = "active"
	IncidentResolved IncidentStatus = "resolved"
	IncidentRevoked  IncidentStatus = "revoked"
)

type Incident struct {
	ID          IncidentID     `json:"id"`
	DatasetID   DatasetID      `json:"datasetId"`
	EdgeIDs     []EdgeID       `json:"edgeIds"`
	Status      IncidentStatus `json:"status"`
	Window      TimeWindow     `json:"window"`
	SpeedFactor float64        `json:"speedFactor"`
	Closed      bool           `json:"closed"`
	Priority    int            `json:"priority"`
	Description string         `json:"description"`
	Version     int64          `json:"version"`
	CreatedAt   time.Time      `json:"createdAt"`
}

func (i Incident) Validate() error {
	if i.ID == "" || i.DatasetID == "" || len(i.EdgeIDs) == 0 {
		return errors.New("incident id, dataset and edges are required")
	}
	if !i.Window.Valid() || i.SpeedFactor < 0 {
		return errors.New("invalid incident window or speed factor")
	}
	return nil
}
func (i Incident) Applies(edge EdgeID, at time.Time) bool {
	if i.Status != IncidentActive || !i.Window.Contains(at) {
		return false
	}
	for _, id := range i.EdgeIDs {
		if id == edge {
			return true
		}
	}
	return false
}

type RouteRequest struct {
	DatasetID    DatasetID `json:"datasetId"`
	From         NodeID    `json:"from"`
	To           NodeID    `json:"to"`
	Mode         Mode      `json:"mode"`
	Departure    time.Time `json:"departure"`
	VehicleClass string    `json:"vehicleClass"`
	AvoidTolls   bool      `json:"avoidTolls"`
	MaxSlope     float64   `json:"maxSlope"`
}

func (r RouteRequest) Validate() error {
	if r.DatasetID == "" || r.From == "" || r.To == "" {
		return errors.New("datasetId, from and to are required")
	}
	if !r.Mode.Valid() {
		return errors.New("valid mode is required")
	}
	if r.From == r.To {
		return errors.New("origin and destination must differ")
	}
	if r.Departure.IsZero() {
		return errors.New("departure is required")
	}
	return nil
}

type RouteLeg struct {
	EdgeID       EdgeID       `json:"edgeId"`
	From         NodeID       `json:"from"`
	To           NodeID       `json:"to"`
	Seconds      float64      `json:"seconds"`
	LengthMeters float64      `json:"lengthMeters"`
	TollCents    int          `json:"tollCents"`
	IncidentIDs  []IncidentID `json:"incidentIds,omitempty"`
}
type RouteResult struct {
	DatasetID       DatasetID  `json:"datasetId"`
	SnapshotVersion int64      `json:"snapshotVersion"`
	Algorithm       string     `json:"algorithm"`
	Departure       time.Time  `json:"departure"`
	Arrival         time.Time  `json:"arrival"`
	TotalSeconds    float64    `json:"totalSeconds"`
	TotalMeters     float64    `json:"totalMeters"`
	TotalTollCents  int        `json:"totalTollCents"`
	Legs            []RouteLeg `json:"legs"`
	Explanation     []string   `json:"explanation"`
}

type JobType string

const (
	JobODMatrix   JobType = "od_matrix"
	JobIsochrone  JobType = "isochrone"
	JobCentrality JobType = "centrality"
)

type JobStatus string

const (
	JobQueued    JobStatus = "queued"
	JobRunning   JobStatus = "running"
	JobSucceeded JobStatus = "succeeded"
	JobFailed    JobStatus = "failed"
	JobCancelled JobStatus = "cancelled"
)

type AnalysisJob struct {
	ID          JobID          `json:"id"`
	Type        JobType        `json:"type"`
	DatasetID   DatasetID      `json:"datasetId"`
	Status      JobStatus      `json:"status"`
	Progress    int            `json:"progress"`
	Parameters  map[string]any `json:"parameters"`
	Result      map[string]any `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	StartedAt   *time.Time     `json:"startedAt,omitempty"`
	CompletedAt *time.Time     `json:"completedAt,omitempty"`
	Checkpoint  int            `json:"checkpoint"`
}

func (j AnalysisJob) Validate() error {
	if j.ID == "" || j.DatasetID == "" {
		return errors.New("job id and dataset id are required")
	}
	if j.Type != JobODMatrix && j.Type != JobIsochrone && j.Type != JobCentrality {
		return errors.New("unsupported job type")
	}
	return nil
}

type EdgeCost struct {
	Edge        Edge
	Seconds     float64
	IncidentIDs []IncidentID
}

func EffectiveCost(edge Edge, incidents []Incident, request RouteRequest, at time.Time) (EdgeCost, bool) {
	if !edge.Supports(request.Mode, at, request.VehicleClass) || (request.AvoidTolls && edge.TollCents > 0) || (request.MaxSlope > 0 && math.Abs(edge.SlopePercent) > request.MaxSlope) {
		return EdgeCost{}, false
	}
	seconds := edge.BaseSeconds
	applied := make([]IncidentID, 0)
	active := make([]Incident, 0)
	for _, incident := range incidents {
		if incident.Applies(edge.ID, at) {
			active = append(active, incident)
		}
	}
	sort.Slice(active, func(i, j int) bool { return active[i].Priority > active[j].Priority })
	for _, incident := range active {
		applied = append(applied, incident.ID)
		if incident.Closed {
			return EdgeCost{}, false
		}
		if incident.SpeedFactor > 0 {
			seconds /= incident.SpeedFactor
		}
	}
	return EdgeCost{Edge: edge, Seconds: seconds, IncidentIDs: applied}, true
}
