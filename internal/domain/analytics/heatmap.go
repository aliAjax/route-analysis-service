package analytics

import (
	"errors"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

type Cell struct {
	Row            int     `json:"row"`
	Column         int     `json:"column"`
	Count          float64 `json:"count"`
	AverageSeconds float64 `json:"averageSeconds"`
}
type Heatmap struct {
	mu      sync.RWMutex
	rows    int
	columns int
	cells   map[[2]int]*Cell
	min     model.Point
	max     model.Point
}

func NewHeatmap(min, max model.Point, rows, columns int) (*Heatmap, error) {
	if !min.Valid() || !max.Valid() || rows < 1 || columns < 1 || max.Lat <= min.Lat || max.Lon <= min.Lon {
		return nil, errors.New("invalid heatmap bounds")
	}
	return &Heatmap{rows: rows, columns: columns, cells: map[[2]int]*Cell{}, min: min, max: max}, nil
}
func (h *Heatmap) locate(point model.Point) (int, int, bool) {
	if !point.Valid() || point.Lat < h.min.Lat || point.Lat > h.max.Lat || point.Lon < h.min.Lon || point.Lon > h.max.Lon {
		return 0, 0, false
	}
	row := int((point.Lat - h.min.Lat) / (h.max.Lat - h.min.Lat) * float64(h.rows))
	col := int((point.Lon - h.min.Lon) / (h.max.Lon - h.min.Lon) * float64(h.columns))
	if row >= h.rows {
		row = h.rows - 1
	}
	if col >= h.columns {
		col = h.columns - 1
	}
	return row, col, true
}
func (h *Heatmap) Add(point model.Point, seconds float64) error {
	if math.IsNaN(seconds) || seconds < 0 {
		return errors.New("seconds must be finite and non-negative")
	}
	row, col, ok := h.locate(point)
	if !ok {
		return errors.New("point outside heatmap")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	key := [2]int{row, col}
	cell := h.cells[key]
	if cell == nil {
		cell = &Cell{Row: row, Column: col}
		h.cells[key] = cell
	}
	cell.Count++
	cell.AverageSeconds += (seconds - cell.AverageSeconds) / cell.Count
	return nil
}
func (h *Heatmap) Cells() []Cell {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]Cell, 0, len(h.cells))
	for _, cell := range h.cells {
		out = append(out, *cell)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Row == out[j].Row {
			return out[i].Column < out[j].Column
		}
		return out[i].Row < out[j].Row
	})
	return out
}
func (h *Heatmap) Bounds() map[string]model.Point {
	return map[string]model.Point{"min": h.min, "max": h.max}
}
func (h *Heatmap) Reset() { h.mu.Lock(); defer h.mu.Unlock(); h.cells = map[[2]int]*Cell{} }

type WindowCounter struct {
	mu     sync.RWMutex
	events map[string][]time.Time
}

func NewWindowCounter() *WindowCounter { return &WindowCounter{events: map[string][]time.Time{}} }
func (c *WindowCounter) Record(key string, at time.Time) error {
	if key == "" || at.IsZero() {
		return errors.New("key and timestamp required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events[key] = append(c.events[key], at)
	return nil
}
func (c *WindowCounter) Count(key string, from, to time.Time) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	count := 0
	for _, at := range c.events[key] {
		if !at.Before(from) && !at.After(to) {
			count++
		}
	}
	return count
}
func (c *WindowCounter) Prune(before time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for key, items := range c.events {
		kept := items[:0]
		for _, at := range items {
			if at.Before(before) {
				removed++
			} else {
				kept = append(kept, at)
			}
		}
		if len(kept) == 0 {
			delete(c.events, key)
		} else {
			c.events[key] = kept
		}
	}
	return removed
}
