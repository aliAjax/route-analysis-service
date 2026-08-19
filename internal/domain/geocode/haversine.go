package geocode

import (
	"math"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// DistanceMeters returns the great-circle distance between two points.
func DistanceMeters(a, b model.Point) float64 {
	return a.DistanceMeters(b)
}

// BearingDegrees returns the initial bearing from a to b in degrees.
func BearingDegrees(a, b model.Point) float64 {
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180
	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)
	return math.Mod(math.Atan2(y, x)*180/math.Pi+360, 360)
}

// Destination returns the point reached by travelling distance meters from p
// along the given bearing in degrees.
func Destination(p model.Point, bearingDegrees, meters float64) model.Point {
	radius := 6371000.0
	lat1 := p.Lat * math.Pi / 180
	lon1 := p.Lon * math.Pi / 180
	bearing := bearingDegrees * math.Pi / 180
	angular := meters / radius
	lat2 := math.Asin(math.Sin(lat1)*math.Cos(angular) + math.Cos(lat1)*math.Sin(angular)*math.Cos(bearing))
	lon2 := lon1 + math.Atan2(math.Sin(bearing)*math.Sin(angular)*math.Cos(lat1), math.Cos(angular)-math.Sin(lat1)*math.Sin(lat2))
	return model.Point{Lat: lat2 * 180 / math.Pi, Lon: lon2 * 180 / math.Pi}
}
