package network

import (
	"errors"
	"sort"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// Path is an ordered sequence of edge ids.
type Path struct {
	Edges []model.EdgeID `json:"edges"`
}

// Reverse returns the reversed edge list.
func (p Path) Reverse() Path {
	out := make([]model.EdgeID, len(p.Edges))
	for i, id := range p.Edges {
		out[len(p.Edges)-1-i] = id
	}
	return Path{Edges: out}
}

// Len returns the number of edges.
func (p Path) Len() int { return len(p.Edges) }

// Validate ensures the path has no empty edge ids.
func (p Path) Validate() error {
	for _, id := range p.Edges {
		if id == "" {
			return errors.New("path contains empty edge id")
		}
	}
	return nil
}

// PathSet stores unique paths and rejects duplicates.
type PathSet struct {
	seen map[string]bool
}

func NewPathSet() *PathSet { return &PathSet{seen: map[string]bool{}} }

// Add inserts a path if it has not been seen before.
func (s *PathSet) Add(p Path) bool {
	key := pathSignature(p)
	if s.seen[key] {
		return false
	}
	s.seen[key] = true
	return true
}

func pathSignature(p Path) string {
	out := ""
	for _, id := range p.Edges {
		out += string(id) + "/"
	}
	return out
}

// RankPaths sorts paths by total seconds and keeps the best k.
func RankPaths(paths []model.RouteResult, k int) []model.RouteResult {
	if k < 1 {
		return nil
	}
	sorted := append([]model.RouteResult(nil), paths...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].TotalSeconds < sorted[j].TotalSeconds
	})
	if len(sorted) > k {
		sorted = sorted[:k]
	}
	return sorted
}
