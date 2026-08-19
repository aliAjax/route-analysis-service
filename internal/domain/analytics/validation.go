package analytics

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

type Issue struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Entity   string `json:"entity"`
	Message  string `json:"message"`
}
type Report struct {
	DatasetID model.DatasetID `json:"datasetId"`
	NodeCount int             `json:"nodeCount"`
	EdgeCount int             `json:"edgeCount"`
	Issues    []Issue         `json:"issues"`
	Valid     bool            `json:"valid"`
	CheckedAt time.Time       `json:"checkedAt"`
}

func ValidateGraph(dataset model.DatasetID, nodes []model.Node, edges []model.Edge) Report {
	report := Report{DatasetID: dataset, NodeCount: len(nodes), EdgeCount: len(edges), Issues: []Issue{}, CheckedAt: time.Now().UTC()}
	nodeSet := map[model.NodeID]model.Node{}
	for _, node := range nodes {
		if _, ok := nodeSet[node.ID]; ok {
			report.Issues = append(report.Issues, Issue{"duplicate_node", "error", string(node.ID), "node identifier is duplicated"})
		}
		nodeSet[node.ID] = node
		if err := node.Validate(); err != nil {
			report.Issues = append(report.Issues, Issue{"invalid_node", "error", string(node.ID), err.Error()})
		}
	}
	edgeSet := map[model.EdgeID]bool{}
	for _, edge := range edges {
		if edgeSet[edge.ID] {
			report.Issues = append(report.Issues, Issue{"duplicate_edge", "error", string(edge.ID), "edge identifier is duplicated"})
		}
		edgeSet[edge.ID] = true
		if _, ok := nodeSet[edge.From]; !ok {
			report.Issues = append(report.Issues, Issue{"missing_origin", "error", string(edge.ID), "origin node is missing"})
		}
		if _, ok := nodeSet[edge.To]; !ok {
			report.Issues = append(report.Issues, Issue{"missing_destination", "error", string(edge.ID), "destination node is missing"})
		}
		if err := edge.Validate(); err != nil {
			report.Issues = append(report.Issues, Issue{"invalid_edge", "error", string(edge.ID), err.Error()})
		}
		if math.IsNaN(edge.LengthMeters) || math.IsInf(edge.LengthMeters, 0) {
			report.Issues = append(report.Issues, Issue{"invalid_length", "error", string(edge.ID), "length is not finite"})
		}
	}
	report.Valid = len(report.Issues) == 0
	return report
}
func Connectivity(nodes []model.Node, edges []model.Edge) map[model.NodeID]int {
	adj := map[model.NodeID][]model.NodeID{}
	for _, node := range nodes {
		adj[node.ID] = nil
	}
	for _, edge := range edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
		adj[edge.To] = append(adj[edge.To], edge.From)
	}
	components := map[model.NodeID]int{}
	component := 0
	for _, node := range nodes {
		if _, seen := components[node.ID]; seen {
			continue
		}
		component++
		queue := []model.NodeID{node.ID}
		components[node.ID] = component
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, next := range adj[current] {
				if _, seen := components[next]; !seen {
					components[next] = component
					queue = append(queue, next)
				}
			}
		}
	}
	return components
}
func TopNodes(nodes []model.Node, edges []model.Edge, limit int) []model.NodeID {
	if limit < 1 {
		return nil
	}
	scores := map[model.NodeID]int{}
	for _, edge := range edges {
		scores[edge.From]++
		scores[edge.To]++
	}
	ids := make([]model.NodeID, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.ID)
	}
	sort.Slice(ids, func(i, j int) bool {
		if scores[ids[i]] == scores[ids[j]] {
			return ids[i] < ids[j]
		}
		return scores[ids[i]] > scores[ids[j]]
	})
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids
}
// ValidateDatasetState reports whether a dataset status is allowed to run graph
// validation. The production build returns nil unconditionally.
func ValidateDatasetState(status model.DatasetStatus) error {
	return nil
}

// DatasetValidationEligible reports whether graph validation is available for
// the given status.
func DatasetValidationEligible(status model.DatasetStatus) bool {
	return true
}

func ValidateDemand(pairs []DemandPair) error {
	for _, pair := range pairs {
		if pair.From == "" || pair.To == "" {
			return errors.New("demand endpoints are required")
		}
		if pair.Count < 0 || math.IsNaN(pair.Count) || math.IsInf(pair.Count, 0) {
			return fmt.Errorf("invalid demand for %s to %s", pair.From, pair.To)
		}
	}
	return nil
}
