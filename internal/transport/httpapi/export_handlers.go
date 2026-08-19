package httpapi

import (
	"net/http"
	"strings"

	"github.com/example/route-analysis-service/internal/domain/model"
)

func (h *Handler) networkReport(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.PathValue("id"))
	report, err := h.service.BuildNetworkReport(r.Context(), id)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if r.URL.Query().Get("format") == "csv" {
		content, _ := report.ToCSV()
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
		return
	}
	respond(w, report, nil)
}

func (h *Handler) incidentReport(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.PathValue("id"))
	report, err := h.service.BuildIncidentReport(r.Context(), id)
	if err != nil {
		respond(w, nil, err)
		return
	}
	respond(w, report, nil)
}

func (h *Handler) odExport(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.PathValue("id"))
	origins := splitNodeList(r.URL.Query().Get("origins"))
	destinations := splitNodeList(r.URL.Query().Get("destinations"))
	if len(origins) == 0 || len(destinations) == 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "origins and destinations are required")
		return
	}
	content, err := h.service.ExportODMatrix(r.Context(), id, origins, destinations)
	if err != nil {
		respond(w, nil, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func splitNodeList(raw string) []model.NodeID {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]model.NodeID, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, model.NodeID(trimmed))
		}
	}
	return out
}
