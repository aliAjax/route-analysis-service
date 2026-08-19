package httpapi

import (
	"net/http"
	"strings"

	"github.com/example/route-analysis-service/internal/domain/model"
)

func (h *Handler) validateDatasetsBatch(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("datasetIds")
	if raw == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "datasetIds is required")
		return
	}
	parts := strings.Split(raw, ",")
	ids := make([]model.DatasetID, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			ids = append(ids, model.DatasetID(trimmed))
		}
	}
	if len(ids) == 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "datasetIds is required")
		return
	}
	results, err := h.service.ValidateDatasetsBatch(r.Context(), ids)
	respond(w, results, err)
}
