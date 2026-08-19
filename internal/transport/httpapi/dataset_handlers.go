package httpapi

import (
	"net/http"
	"strconv"

	"github.com/example/route-analysis-service/internal/domain/model"
)

func (h *Handler) getDataset(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.PathValue("id"))
	item, err := h.service.GetDataset(r.Context(), id)
	respond(w, item, err)
}

func (h *Handler) summariseDataset(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.PathValue("id"))
	summary, err := h.service.SummariseDataset(r.Context(), id)
	respond(w, summary, err)
}

func (h *Handler) validateDataset(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.PathValue("id"))
	report, err := h.service.ValidateDataset(r.Context(), id)
	respond(w, report, err)
}

func (h *Handler) clusterNodes(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.PathValue("id"))
	radius, _ := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
	clusters, err := h.service.ClusterNodes(r.Context(), id, radius)
	respond(w, clusters, err)
}
