package r10

import (
	"testing"

	"github.com/example/route-analysis-service/internal/domain/geocode"
	"github.com/example/route-analysis-service/internal/domain/model"
)

func TestClusterAndDistanceResultsDoNotAlias(t *testing.T) {
	nodes := map[model.NodeID]model.Point{
		"a": {Lat: 35.6812, Lon: 139.7671},
		"b": {Lat: 35.6813, Lon: 139.7672},
		"c": {Lat: 35.6900, Lon: 139.7800},
		"d": {Lat: 35.6901, Lon: 139.7801},
	}
	clusters := geocode.ClusterNodes(nodes, 100)
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
	if len(clusters[0].Nodes) == 0 || len(clusters[1].Nodes) == 0 {
		t.Fatalf("clusters should not be empty: %v", clusters)
	}
	clusters[0].Nodes[0] = "mutated"
	if clusters[1].Nodes[0] == "mutated" {
		t.Fatalf("cluster node lists share mutable backing: %v / %v", clusters[0].Nodes, clusters[1].Nodes)
	}

	origin := model.Point{Lat: 35.6812, Lon: 139.7671}
	points := []model.Point{{Lat: 35.6813, Lon: 139.7672}, {Lat: 35.6900, Lon: 139.7800}}
	first := geocode.Distances(origin, points)
	second := geocode.Distances(origin, points)
	if len(first) != 2 || len(second) != 2 {
		t.Fatal("distance lists invalid")
	}
	first[0] = -999
	if second[0] == -999 {
		t.Fatal("distance lists share mutable backing")
	}
}
