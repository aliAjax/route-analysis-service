package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/example/route-analysis-service/internal/application"
	"github.com/example/route-analysis-service/internal/domain/model"
	"github.com/example/route-analysis-service/internal/infrastructure/memory"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	service *application.Service
	logger  *slog.Logger
}

func NewHandler(s *application.Service, l *slog.Logger) *Handler {
	return &Handler{service: s, logger: l}
}
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /readyz", h.health)
	mux.HandleFunc("GET /api/v1/datasets", h.listDatasets)
	mux.HandleFunc("POST /api/v1/datasets", h.createDataset)
	mux.HandleFunc("POST /api/v1/datasets/{id}/import", h.importGraph)
	mux.HandleFunc("POST /api/v1/route", h.route)
	mux.HandleFunc("POST /api/v1/k-routes", h.kRoutes)
	mux.HandleFunc("POST /api/v1/isochrone", h.isochrone)
	mux.HandleFunc("GET /api/v1/incidents", h.listIncidents)
	mux.HandleFunc("POST /api/v1/incidents", h.createIncident)
	mux.HandleFunc("POST /api/v1/incidents/{id}/activate", h.activateIncident)
	mux.HandleFunc("POST /api/v1/jobs", h.createJob)
	mux.HandleFunc("GET /api/v1/jobs/{id}", h.getJob)
	mux.HandleFunc("GET /api/v1/jobs", h.listJobs)
	mux.HandleFunc("GET /api/v1/datasets/{id}", h.getDataset)
	mux.HandleFunc("GET /api/v1/datasets/{id}/summary", h.summariseDataset)
	mux.HandleFunc("GET /api/v1/datasets/{id}/validate", h.validateDataset)
	mux.HandleFunc("POST /api/v1/datasets/validate-batch", h.validateDatasetsBatch)
	mux.HandleFunc("GET /api/v1/datasets/{id}/clusters", h.clusterNodes)
	mux.HandleFunc("GET /api/v1/geocode/nearest", h.nearestNode)
	mux.HandleFunc("GET /api/v1/incidents/conflicts", h.incidentConflicts)
	mux.HandleFunc("POST /api/v1/incidents/resolve-expired", h.resolveExpiredIncidents)
	mux.HandleFunc("GET /api/v1/datasets/{id}/reports/network", h.networkReport)
	mux.HandleFunc("GET /api/v1/datasets/{id}/reports/incidents", h.incidentReport)
	mux.HandleFunc("GET /api/v1/datasets/{id}/od-export", h.odExport)
	mux.HandleFunc("POST /api/v1/quota/policies", h.registerQuota)
	mux.HandleFunc("GET /api/v1/quota/status", h.quotaStatus)
	mux.Handle("GET /", http.FileServer(http.Dir("./web")))
	return h.middleware(mux)
}
func (h *Handler) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("X-Request-ID", requestID(r))
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			if r.Header.Get("Content-Type") != "" && !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "content type must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
		h.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
	})
}
func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "route-analysis-service", "time": time.Now().UTC()})
}
func (h *Handler) listDatasets(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListDatasets(r.Context())
	respond(w, items, err)
}
func (h *Handler) createDataset(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Tags        map[string]string `json:"tags"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := h.service.CreateDataset(r.Context(), input.Name, input.Description, input.Tags)
	respond(w, item, err)
}
func (h *Handler) importGraph(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Nodes []model.Node `json:"nodes"`
		Edges []model.Edge `json:"edges"`
	}
	if !decode(w, r, &input) {
		return
	}
	id := model.DatasetID(r.PathValue("id"))
	for i := range input.Nodes {
		input.Nodes[i].DatasetID = id
	}
	for i := range input.Edges {
		input.Edges[i].DatasetID = id
	}
	respond(w, map[string]any{"datasetId": id, "nodes": len(input.Nodes), "edges": len(input.Edges)}, h.service.ImportGraph(r.Context(), id, input.Nodes, input.Edges))
}
func (h *Handler) route(w http.ResponseWriter, r *http.Request) {
	var req model.RouteRequest
	if !decode(w, r, &req) {
		return
	}
	result, err := h.service.Route(r.Context(), req)
	respond(w, result, err)
}
func (h *Handler) kRoutes(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Request model.RouteRequest `json:"request"`
		K       int                `json:"k"`
	}
	if !decode(w, r, &input) {
		return
	}
	result, err := h.service.KRoutes(r.Context(), input.Request, input.K)
	respond(w, result, err)
}
func (h *Handler) isochrone(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Request       model.RouteRequest `json:"request"`
		BudgetSeconds float64            `json:"budgetSeconds"`
	}
	if !decode(w, r, &input) {
		return
	}
	result, err := h.service.Isochrone(r.Context(), input.Request, input.BudgetSeconds)
	respond(w, map[string]any{"reachable": result, "count": len(result)}, err)
}
func (h *Handler) listIncidents(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.URL.Query().Get("datasetId"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "datasetId is required")
		return
	}
	items, err := h.service.ListIncidents(r.Context(), id)
	respond(w, items, err)
}
func (h *Handler) createIncident(w http.ResponseWriter, r *http.Request) {
	var incident model.Incident
	if !decode(w, r, &incident) {
		return
	}
	created, err := h.service.CreateIncident(r.Context(), incident)
	respond(w, created, err)
}
func (h *Handler) activateIncident(w http.ResponseWriter, r *http.Request) {
	dataset := model.DatasetID(r.URL.Query().Get("datasetId"))
	version, err := strconv.ParseInt(r.Header.Get("If-Match"), 10, 64)
	if dataset == "" || err != nil {
		writeError(w, http.StatusPreconditionRequired, "precondition_required", "datasetId and numeric If-Match header are required")
		return
	}
	respond(w, map[string]any{"id": r.PathValue("id"), "status": "active"}, h.service.ActivateIncident(r.Context(), model.IncidentID(r.PathValue("id")), dataset, version))
}
func (h *Handler) createJob(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type       model.JobType   `json:"type"`
		DatasetID  model.DatasetID `json:"datasetId"`
		Parameters map[string]any  `json:"parameters"`
	}
	if !decode(w, r, &input) {
		return
	}
	item, err := h.service.CreateJob(r.Context(), input.Type, input.DatasetID, input.Parameters)
	respond(w, item, err)
}
func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetJob(r.Context(), model.JobID(r.PathValue("id")))
	respond(w, item, err)
}
func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	id := model.DatasetID(r.URL.Query().Get("datasetId"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "datasetId is required")
		return
	}
	items, err := h.service.ListJobs(r.Context(), id)
	respond(w, items, err)
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return false
	}
	return true
}
func respond(w http.ResponseWriter, value any, err error) {
	if err == nil {
		writeJSON(w, http.StatusOK, value)
		return
	}
	switch {
	case errors.Is(err, memory.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, memory.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, contextCanceled):
		writeError(w, http.StatusRequestTimeout, "cancelled", err.Error())
	default:
		writeError(w, http.StatusBadRequest, "operation_failed", err.Error())
	}
}

var contextCanceled = errors.New("context canceled")

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
