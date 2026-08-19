package network

import (
	"github.com/example/route-analysis-service/internal/domain/model"
)

// Connectivity describes weakly-connected component labels.
type Connectivity struct {
	labels map[model.NodeID]int
	count  int
}

// ComputeConnectivity labels weakly-connected components over the topology.
func ComputeConnectivity(t *Topology) *Connectivity {
	labels := map[model.NodeID]int{}
	next := 0
	for _, id := range t.NodesSorted() {
		if _, seen := labels[id]; seen {
			continue
		}
		next++
		queue := []model.NodeID{id}
		labels[id] = next
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, edge := range append(append([]model.Edge{}, t.Outgoing[current]...), t.Incoming[current]...) {
				neighbor := edge.To
				if neighbor == current {
					neighbor = edge.From
				}
				if _, seen := labels[neighbor]; !seen {
					labels[neighbor] = next
					queue = append(queue, neighbor)
				}
			}
		}
	}
	return &Connectivity{labels: labels, count: next}
}

// Component returns the component id of a node and whether it was seen.
func (c *Connectivity) Component(id model.NodeID) (int, bool) {
	value, ok := c.labels[id]
	return value, ok
}

// ComponentCount returns the number of weakly-connected components.
func (c *Connectivity) ComponentCount() int { return c.count }

// SameComponent reports whether two nodes are weakly connected.
func (c *Connectivity) SameComponent(a, b model.NodeID) bool {
	left, leftOK := c.labels[a]
	right, rightOK := c.labels[b]
	return leftOK && rightOK && left == right
}
