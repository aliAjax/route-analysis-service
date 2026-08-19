package incident

import (
	"errors"
	"sort"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// Conflict describes two incidents whose effect windows and edge sets overlap.
type Conflict struct {
	Left     model.IncidentID `json:"left"`
	Right    model.IncidentID `json:"right"`
	EdgeIDs  []model.EdgeID   `json:"edgeIds"`
	Overlaps bool             `json:"overlaps"`
}

// Resolver evaluates priority conflicts between active incidents.
type Resolver struct {
	Now func() time.Time
}

func NewResolver(now func() time.Time) *Resolver {
	if now == nil {
		now = time.Now
	}
	return &Resolver{Now: now}
}

// SortByPriority returns incidents ordered by descending priority then by
// ascending creation time for deterministic evaluation.
func (r *Resolver) SortByPriority(items []model.Incident) []model.Incident {
	out := append([]model.Incident(nil), items...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].Priority > out[j].Priority
	})
	return out
}

// Conflicts returns pairs of active incidents that touch at least one shared
// edge inside an overlapping time window.
func (r *Resolver) Conflicts(items []model.Incident, at time.Time) []Conflict {
	active := make([]model.Incident, 0, len(items))
	for _, item := range items {
		if item.Status == model.IncidentActive && item.Window.Contains(at) {
			active = append(active, item)
		}
	}
	out := make([]Conflict, 0)
	for i := 0; i < len(active); i++ {
		for j := i + 1; j < len(active); j++ {
			left, right := active[i], active[j]
			shared := sharedEdges(left, right)
			if len(shared) == 0 {
				continue
			}
			out = append(out, Conflict{
				Left:     left.ID,
				Right:    right.ID,
				EdgeIDs:  shared,
				Overlaps: windowsOverlap(left.Window, right.Window),
			})
		}
	}
	return out
}

// Dominant returns the highest-priority incident affecting an edge at a time.
func (r *Resolver) Dominant(items []model.Incident, edge model.EdgeID, at time.Time) (model.Incident, bool) {
	ordered := r.SortByPriority(items)
	for _, item := range ordered {
		if item.Status == model.IncidentActive && item.Window.Contains(at) {
			for _, id := range item.EdgeIDs {
				if id == edge {
					return item, true
				}
			}
		}
	}
	return model.Incident{}, false
}

func sharedEdges(a, b model.Incident) []model.EdgeID {
	index := make(map[model.EdgeID]bool, len(a.EdgeIDs))
	for _, id := range a.EdgeIDs {
		index[id] = true
	}
	out := make([]model.EdgeID, 0)
	for _, id := range b.EdgeIDs {
		if index[id] {
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func windowsOverlap(a, b model.TimeWindow) bool {
	if a.Start.IsZero() || b.Start.IsZero() {
		return true
	}
	return !a.End.Before(b.Start) && !b.End.Before(a.Start)
}

var ErrNoDominant = errors.New("no active incident applies to the edge")
