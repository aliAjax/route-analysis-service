package httpapi

import (
	"net/http"
	"time"

	"github.com/example/route-analysis-service/internal/domain/model"
)

func (h *Handler) incidentConflicts(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.URL.Query().Get("datasetId"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "datasetId is required")
		return
	}
	at := time.Now().UTC()
	if raw := r.URL.Query().Get("at"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "at must be RFC3339")
			return
		}
		at = parsed
	}
	conflicts, err := h.service.IncidentConflicts(r.Context(), id, at)
	if err != nil {
		respond(w, nil, err)
		return
	}
	var shared []model.EdgeID
	for i := range conflicts {
		shared = shared[:0]
		shared = append(shared, conflicts[i].EdgeIDs...)
		conflicts[i].EdgeIDs = shared
	}
	respond(w, conflicts, nil)
}

func (h *Handler) resolveExpiredIncidents(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.URL.Query().Get("datasetId"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "datasetId is required")
		return
	}
	at := time.Now().UTC()
	if raw := r.URL.Query().Get("at"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "at must be RFC3339")
			return
		}
		at = parsed
	}
	ids, err := h.service.ResolveExpiredIncidents(r.Context(), id, at)
	respond(w, map[string]any{"resolved": ids, "count": len(ids)}, err)
}
