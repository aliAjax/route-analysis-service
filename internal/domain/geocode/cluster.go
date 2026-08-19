package geocode

import (
	"sort"

	"github.com/example/route-analysis-service/internal/domain/model"
)

// Cluster is a group of node ids that are geographically close.
type Cluster struct {
	Centroid model.Point    `json:"centroid"`
	Nodes    []model.NodeID `json:"nodes"`
}

// ClusterNodes greedily groups nodes whose pairwise distance is at most radius.
func ClusterNodes(nodes map[model.NodeID]model.Point, radiusMeters float64) []Cluster {
	if radiusMeters <= 0 {
		radiusMeters = 500
	}
	ids := make([]model.NodeID, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	remaining := make(map[model.NodeID]bool, len(ids))
	for _, id := range ids {
		remaining[id] = true
	}
	clusters := make([]Cluster, 0)
	for _, seed := range ids {
		if !remaining[seed] {
			continue
		}
		clusterNodes := make([]model.NodeID, 0)
		clusterNodes = append(clusterNodes, seed)
		latSum, lonSum := nodes[seed].Lat, nodes[seed].Lon
		for _, candidate := range ids {
			if candidate == seed || !remaining[candidate] {
				continue
			}
			if DistanceMeters(nodes[seed], nodes[candidate]) <= radiusMeters {
				clusterNodes = append(clusterNodes, candidate)
				latSum += nodes[candidate].Lat
				lonSum += nodes[candidate].Lon
				remaining[candidate] = false
			}
		}
		remaining[seed] = false
		n := float64(len(clusterNodes))
		cluster := Cluster{Nodes: clusterNodes, Centroid: model.Point{Lat: latSum / n, Lon: lonSum / n}}
		clusters = append(clusters, cluster)
	}
	return clusters
}
