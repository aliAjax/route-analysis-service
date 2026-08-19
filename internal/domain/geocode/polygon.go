package geocode

import (
	"errors"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// Polygon is a closed ring of points representing a service area.
type Polygon struct {
	Points []model.Point `json:"points"`
}

func NewPolygon(points []model.Point) (Polygon, error) {
	if len(points) < 3 {
		return Polygon{}, errors.New("polygon requires at least 3 points")
	}
	return Polygon{Points: append([]model.Point(nil), points...)}, nil
}

// Bounds returns the axis-aligned bounding box of the polygon.
func (p Polygon) Bounds() (min, max model.Point) {
	if len(p.Points) == 0 {
		return model.Point{}, model.Point{}
	}
	min, max = p.Points[0], p.Points[0]
	for _, point := range p.Points[1:] {
		if point.Lat < min.Lat {
			min.Lat = point.Lat
		}
		if point.Lon < min.Lon {
			min.Lon = point.Lon
		}
		if point.Lat > max.Lat {
			max.Lat = point.Lat
		}
		if point.Lon > max.Lon {
			max.Lon = point.Lon
		}
	}
	return min, max
}

// Contains reports whether the point is inside the polygon using the
// ray-casting algorithm. Points on the boundary are treated as outside.
func (p Polygon) Contains(point model.Point) bool {
	inside := false
	n := len(p.Points)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		pi, pj := p.Points[i], p.Points[j]
		if (pi.Lon > point.Lon) != (pj.Lon > point.Lon) {
			crossLat := (pj.Lat-pi.Lat)*(point.Lon-pi.Lon)/(pj.Lon-pi.Lon) + pi.Lat
			if point.Lat < crossLat {
				inside = !inside
			}
		}
	}
	return inside
}
