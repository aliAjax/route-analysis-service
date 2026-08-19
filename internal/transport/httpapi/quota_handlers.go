package httpapi

import (
	"net/http"

	"github.com/example/route-analysis-service/internal/domain/quota"
)

func (h *Handler) registerQuota(w http.ResponseWriter, r *http.Request) {
	var policy quota.Policy
	if !decode(w, r, &policy) {
		return
	}
	err := h.service.RegisterQuotaPolicy(r.Context(), policy)
	respond(w, map[string]any{"tenant": policy.Tenant}, err)
}

func (h *Handler) quotaStatus(w http.ResponseWriter, r *http.Request) {
	tenant := r.URL.Query().Get("tenant")
	if tenant == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "tenant is required")
		return
	}
	respond(w, h.service.QuotaStatus(r.Context(), tenant), nil)
}
