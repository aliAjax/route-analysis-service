package routing

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

var ErrNoRoute = errors.New("no route found")

type Snapshot struct {
	Nodes     map[model.NodeID]model.Node
	Edges     map[model.EdgeID]model.Edge
	Outgoing  map[model.NodeID][]model.Edge
	Incidents []model.Incident
	Version   int64
}

func NewSnapshot(nodes []model.Node, edges []model.Edge, incidents []model.Incident, version int64) Snapshot {
	s := Snapshot{Nodes: map[model.NodeID]model.Node{}, Edges: map[model.EdgeID]model.Edge{}, Outgoing: map[model.NodeID][]model.Edge{}, Incidents: append([]model.Incident(nil), incidents...), Version: version}
	for _, n := range nodes {
		s.Nodes[n.ID] = n
	}
	for _, e := range edges {
		s.Edges[e.ID] = e
		s.Outgoing[e.From] = append(s.Outgoing[e.From], e)
	}
	for id := range s.Outgoing {
		sort.Slice(s.Outgoing[id], func(i, j int) bool { return s.Outgoing[id][i].ID < s.Outgoing[id][j].ID })
	}
	return s
}

type state struct {
	node     model.NodeID
	cost     float64
	estimate float64
	arrival  time.Time
	previous *state
	edge     model.EdgeCost
	index    int
}
type queue []*state

func (q queue) Len() int           { return len(q) }
func (q queue) Less(i, j int) bool { return q[i].estimate < q[j].estimate }
func (q queue) Swap(i, j int)      { q[i], q[j] = q[j], q[i]; q[i].index = i; q[j].index = j }
func (q *queue) Push(x any)        { v := x.(*state); v.index = len(*q); *q = append(*q, v) }
func (q *queue) Pop() any {
	old := *q
	n := len(old)
	v := old[n-1]
	old[n-1] = nil
	v.index = -1
	*q = old[:n-1]
	return v
}

func (s Snapshot) Route(req model.RouteRequest) (model.RouteResult, error) {
	if err := req.Validate(); err != nil {
		return model.RouteResult{}, err
	}
	if _, ok := s.Nodes[req.From]; !ok {
		return model.RouteResult{}, fmt.Errorf("origin %s: %w", req.From, ErrNoRoute)
	}
	target, ok := s.Nodes[req.To]
	if !ok {
		return model.RouteResult{}, fmt.Errorf("destination %s: %w", req.To, ErrNoRoute)
	}
	initial := &state{node: req.From, arrival: req.Departure}
	open := &queue{initial}
	heap.Init(open)
	best := map[model.NodeID]float64{req.From: 0}
	for open.Len() > 0 {
		current := heap.Pop(open).(*state)
		if known := best[current.node]; current.cost > known+0.000001 {
			continue
		}
		if current.node == req.To {
			return s.result(req, current), nil
		}
		for _, edge := range s.Outgoing[current.node] {
			cost, allowed := model.EffectiveCost(edge, s.Incidents, req, current.arrival)
			if !allowed {
				continue
			}
			candidate := current.cost + cost.Seconds
			if old, seen := best[edge.To]; seen && candidate >= old {
				continue
			}
			arrival := current.arrival.Add(time.Duration(cost.Seconds * float64(time.Second)))
			estimate := candidate + heuristic(s.Nodes[edge.To], target, req.Mode)
			next := &state{node: edge.To, cost: candidate, estimate: estimate, arrival: arrival, previous: current, edge: cost}
			best[edge.To] = candidate
			heap.Push(open, next)
		}
	}
	return model.RouteResult{}, ErrNoRoute
}
func heuristic(from, to model.Node, mode model.Mode) float64 {
	speed := 15.0
	switch mode {
	case model.ModeWalk:
		speed = 1.3
	case model.ModeBike:
		speed = 5.0
	case model.ModeTransit:
		speed = 10
	}
	return from.Location.DistanceMeters(to.Location) / speed
}
func (s Snapshot) result(req model.RouteRequest, end *state) model.RouteResult {
	legs := []model.RouteLeg{}
	for current := end; current.previous != nil; current = current.previous {
		e := current.edge.Edge
		legs = append(legs, model.RouteLeg{EdgeID: e.ID, From: e.From, To: e.To, Seconds: current.edge.Seconds, LengthMeters: e.LengthMeters, TollCents: e.TollCents, IncidentIDs: current.edge.IncidentIDs})
	}
	for i, j := 0, len(legs)-1; i < j; i, j = i+1, j-1 {
		legs[i], legs[j] = legs[j], legs[i]
	}
	meters := 0.0
	toll := 0
	explanation := []string{"A* time-dependent routing", "snapshot version is pinned for query consistency"}
	for _, leg := range legs {
		meters += leg.LengthMeters
		toll += leg.TollCents
		if len(leg.IncidentIDs) > 0 {
			explanation = append(explanation, "active incident adjusted edge "+string(leg.EdgeID))
		}
	}
	return model.RouteResult{DatasetID: req.DatasetID, SnapshotVersion: s.Version, Algorithm: "a-star-time-dependent", Departure: req.Departure, Arrival: end.arrival, TotalSeconds: end.cost, TotalMeters: meters, TotalTollCents: toll, Legs: legs, Explanation: explanation}
}
func (s Snapshot) Reachable(req model.RouteRequest, budgetSeconds float64) (map[model.NodeID]float64, error) {
	if req.DatasetID == "" || req.From == "" || !req.Mode.Valid() || req.Departure.IsZero() {
		return nil, errors.New("datasetId, from, valid mode and departure are required")
	}
	if budgetSeconds <= 0 {
		return nil, errors.New("budget must be positive")
	}
	result := map[model.NodeID]float64{req.From: 0}
	open := &queue{{node: req.From, arrival: req.Departure}}
	heap.Init(open)
	for open.Len() > 0 {
		current := heap.Pop(open).(*state)
		if current.cost > budgetSeconds {
			continue
		}
		if known := result[current.node]; current.cost > known+0.000001 {
			continue
		}
		for _, edge := range s.Outgoing[current.node] {
			cost, ok := model.EffectiveCost(edge, s.Incidents, req, current.arrival)
			if !ok {
				continue
			}
			nextCost := current.cost + cost.Seconds
			if nextCost > budgetSeconds {
				continue
			}
			if old, seen := result[edge.To]; seen && nextCost >= old {
				continue
			}
			result[edge.To] = nextCost
			heap.Push(open, &state{node: edge.To, cost: nextCost, estimate: nextCost, arrival: current.arrival.Add(time.Duration(cost.Seconds * float64(time.Second)))})
		}
	}
	return result, nil
}
func (s Snapshot) KRoutes(req model.RouteRequest, k int) ([]model.RouteResult, error) {
	if k < 1 || k > 10 {
		return nil, errors.New("k must be between 1 and 10")
	}
	first, err := s.Route(req)
	if err != nil {
		return nil, err
	}
	results := []model.RouteResult{first}
	seen := map[string]bool{pathKey(first): true}
	candidates := []model.RouteResult{}
	for len(results) < k {
		previous := results[len(results)-1]
		for _, leg := range previous.Legs {
			modified := s
			copyEdges := map[model.EdgeID]model.Edge{}
			for id, e := range s.Edges {
				copyEdges[id] = e
			}
			delete(copyEdges, leg.EdgeID)
			modified.Edges = copyEdges
			modified.Outgoing = map[model.NodeID][]model.Edge{}
			for _, edge := range copyEdges {
				modified.Outgoing[edge.From] = append(modified.Outgoing[edge.From], edge)
			}
			candidate, routeErr := modified.Route(req)
			if routeErr == nil && !seen[pathKey(candidate)] {
				seen[pathKey(candidate)] = true
				candidates = append(candidates, candidate)
			}
		}
		if len(candidates) == 0 {
			break
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].TotalSeconds < candidates[j].TotalSeconds })
		results = append(results, candidates[0])
		candidates = candidates[1:]
	}
	return results, nil
}
func pathKey(route model.RouteResult) string {
	out := ""
	for _, leg := range route.Legs {
		out += string(leg.EdgeID) + "/"
	}
	return out
}
func (s Snapshot) DegreeCentrality() map[model.NodeID]float64 {
	degrees := map[model.NodeID]float64{}
	for id := range s.Nodes {
		degrees[id] = 0
	}
	for _, edge := range s.Edges {
		degrees[edge.From]++
		degrees[edge.To]++
	}
	n := float64(len(s.Nodes) - 1)
	if n <= 0 {
		return degrees
	}
	for id, value := range degrees {
		degrees[id] = value / n
	}
	return degrees
}
func (s Snapshot) Nearest(ctx context.Context, point model.Point, mode model.Mode) (model.NodeID, float64, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	if !point.Valid() {
		return "", 0, errors.New("invalid point")
	}
	min := math.Inf(1)
	var chosen model.NodeID
	for id, node := range s.Nodes {
		if err := ctx.Err(); err != nil {
			return "", 0, err
		}
		if len(s.Outgoing[id]) == 0 {
			continue
		}
		valid := false
		for _, edge := range s.Outgoing[id] {
			for _, supported := range edge.Modes {
				if supported == mode {
					valid = true
					break
				}
			}
		}
		if !valid {
			continue
		}
		distance := point.DistanceMeters(node.Location)
		if distance < min {
			min = distance
			chosen = id
		}
	}
	if chosen == "" {
		return "", 0, ErrNoRoute
	}
	return chosen, min, nil
}
