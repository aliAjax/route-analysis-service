package network

import (
	"errors"
	"sort"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// Topology is an immutable in-memory view of a directed graph.
type Topology struct {
	Nodes    map[model.NodeID]model.Node
	Edges    map[model.EdgeID]model.Edge
	Outgoing map[model.NodeID][]model.Edge
	Incoming map[model.NodeID][]model.Edge
}

// BuildTopology constructs a topology from node and edge slices.
func BuildTopology(nodes []model.Node, edges []model.Edge) (*Topology, error) {
	t := &Topology{
		Nodes:    map[model.NodeID]model.Node{},
		Edges:    map[model.EdgeID]model.Edge{},
		Outgoing: map[model.NodeID][]model.Edge{},
		Incoming: map[model.NodeID][]model.Edge{},
	}
	for _, node := range nodes {
		if _, exists := t.Nodes[node.ID]; exists {
			return nil, errors.New("duplicate node " + string(node.ID))
		}
		t.Nodes[node.ID] = node
	}
	for _, edge := range edges {
		if _, ok := t.Nodes[edge.From]; !ok {
			return nil, errors.New("edge origin missing: " + string(edge.From))
		}
		if _, ok := t.Nodes[edge.To]; !ok {
			return nil, errors.New("edge destination missing: " + string(edge.To))
		}
		if _, exists := t.Edges[edge.ID]; exists {
			return nil, errors.New("duplicate edge " + string(edge.ID))
		}
		t.Edges[edge.ID] = edge
		t.Outgoing[edge.From] = append(t.Outgoing[edge.From], edge)
		t.Incoming[edge.To] = append(t.Incoming[edge.To], edge)
	}
	for id := range t.Outgoing {
		sort.Slice(t.Outgoing[id], func(i, j int) bool { return t.Outgoing[id][i].ID < t.Outgoing[id][j].ID })
	}
	for id := range t.Incoming {
		sort.Slice(t.Incoming[id], func(i, j int) bool { return t.Incoming[id][i].ID < t.Incoming[id][j].ID })
	}
	return t, nil
}

// Degree returns the in, out and total degree of a node.
func (t *Topology) Degree(id model.NodeID) (in, out, total int) {
	in = len(t.Incoming[id])
	out = len(t.Outgoing[id])
	return in, out, in + out
}

// NodesSorted returns all node ids in stable order.
func (t *Topology) NodesSorted() []model.NodeID {
	ids := make([]model.NodeID, 0, len(t.Nodes))
	for id := range t.Nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// Isolated returns nodes with no incoming or outgoing edges.
func (t *Topology) Isolated() []model.NodeID {
	out := make([]model.NodeID, 0)
	for _, id := range t.NodesSorted() {
		if len(t.Outgoing[id]) == 0 && len(t.Incoming[id]) == 0 {
			out = append(out, id)
		}
	}
	return out
}
