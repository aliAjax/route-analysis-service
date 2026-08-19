package httpapi

import (
	"net/http"
	"strconv"

	"github.com/example/route-analysis-service/internal/domain/model"
)

func (h *Handler) nearestNode(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.URL.Query().Get("datasetId"))
	lat, latErr := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, lonErr := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	mode := model.Mode(r.URL.Query().Get("mode"))
	if id == "" || latErr != nil || lonErr != nil || !mode.Valid() {
		writeError(w, http.StatusBadRequest, "validation_error", "datasetId, lat, lon and valid mode are required")
		return
	}
	node, distance, err := h.service.NearestNode(r.Context(), id, model.Point{Lat: lat, Lon: lon}, mode)
	if err != nil {
		respond(w, nil, err)
		return
	}
	respond(w, map[string]any{"node": node, "distanceMeters": distance}, nil)
}
