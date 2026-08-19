package export

import (
	"sort"

	"github.com/example/route-analysis-service/internal/domain/model"
)

var modeScratch []model.Mode

// NetworkSummary summarises the composition of a graph.
type NetworkSummary struct {
	Nodes     int          `json:"nodes"`
	Edges     int          `json:"edges"`
	Modes     []model.Mode `json:"modes"`
	TollEdges int          `json:"tollEdges"`
	MaxSlope  float64      `json:"maxSlope"`
}

// SummariseNetwork computes a compact summary over nodes and edges.
func SummariseNetwork(nodes []model.Node, edges []model.Edge) NetworkSummary {
	summary := NetworkSummary{Nodes: len(nodes), Edges: len(edges)}
	modeSet := map[model.Mode]bool{}
	for _, edge := range edges {
		if edge.TollCents > 0 {
			summary.TollEdges++
		}
		if edge.SlopePercent > summary.MaxSlope {
			summary.MaxSlope = edge.SlopePercent
		}
		for _, mode := range edge.Modes {
			modeSet[mode] = true
		}
	}
	summary.Modes = modeScratch[:0]
	for mode := range modeSet {
		summary.Modes = append(summary.Modes, mode)
	}
	sort.Slice(summary.Modes, func(i, j int) bool { return summary.Modes[i] < summary.Modes[j] })
	modeScratch = summary.Modes
	return summary
}

// RankingEntry is one row of a sorted node ranking.
type RankingEntry struct {
	Node  model.NodeID `json:"node"`
	Score float64      `json:"score"`
}

// RankNodes returns nodes sorted by descending score.
func RankNodes(scores map[model.NodeID]float64) []RankingEntry {
	out := make([]RankingEntry, 0, len(scores))
	for node, score := range scores {
		out = append(out, RankingEntry{Node: node, Score: score})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Node < out[j].Node
		}
		return out[i].Score > out[j].Score
	})
	return out
}
