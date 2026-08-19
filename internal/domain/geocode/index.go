package geocode

import (
	"math"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// GridIndex is a uniform spatial grid for approximate nearest-point lookups.
type GridIndex struct {
	cellSize float64
	cells    map[[2]int][]gridEntry
}

type gridEntry struct {
	ID    model.NodeID
	Point model.Point
}

// NewGridIndex builds a grid index over node positions.
func NewGridIndex(nodes map[model.NodeID]model.Point, cellSizeMeters float64) *GridIndex {
	if cellSizeMeters <= 0 {
		cellSizeMeters = 1000
	}
	index := &GridIndex{cellSize: cellSizeMeters, cells: map[[2]int][]gridEntry{}}
	for id, point := range nodes {
		key := index.key(point)
		index.cells[key] = append(index.cells[key], gridEntry{ID: id, Point: point})
	}
	return index
}

func (g *GridIndex) key(point model.Point) [2]int {
	latDegrees := g.cellSize / 111320.0
	lonDegrees := g.cellSize / (111320.0 * math.Cos(point.Lat*math.Pi/180))
	return [2]int{int(math.Floor(point.Lat / latDegrees)), int(math.Floor(point.Lon / lonDegrees))}
}

// Nearest returns the closest node within the grid neighbourhood.
func (g *GridIndex) Nearest(point model.Point) (model.NodeID, float64, bool) {
	key := g.key(point)
	best := model.NodeID("")
	bestDistance := math.Inf(1)
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			candidates := g.cells[[2]int{key[0] + dx, key[1] + dy}]
			for _, entry := range candidates {
				distance := DistanceMeters(point, entry.Point)
				if distance < bestDistance {
					bestDistance = distance
					best = entry.ID
				}
			}
		}
	}
	return best, bestDistance, best != ""
}
