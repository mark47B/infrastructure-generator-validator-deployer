package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"orchestrator/app/usecase"
	"orchestrator/internal/domain/entity"
)

type OrchestratorHandler struct {
	jobService        usecase.JobUsecase
	configFileService usecase.ConfigFilesUseCase
	logger            *slog.Logger
	upgrader          websocket.Upgrader

	// метрики
	reqDuration *prometheus.HistogramVec
	reqCount    *prometheus.CounterVec
	errCount    *prometheus.CounterVec
}

func NewOrchestratorHandler(
	jobService usecase.JobUsecase,
	configFileService usecase.ConfigFilesUseCase,
	logger *slog.Logger,
) *OrchestratorHandler {

	reqDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	reqCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed.",
		},
		[]string{"method", "path"},
	)

	errCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP request errors.",
		},
		[]string{"method", "path", "status"},
	)

	prometheus.MustRegister(reqDuration, reqCount, errCount)

	return &OrchestratorHandler{
		jobService:        jobService,
		configFileService: configFileService,
		logger:            logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		reqDuration: reqDuration,
		reqCount:    reqCount,
		errCount:    errCount,
	}
}

// Middleware для метрик
func (h *OrchestratorHandler) withMetrics(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path
		method := r.Method

		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(rw, r)

		duration := time.Since(start).Seconds()
		statusStr := strconv.Itoa(rw.status)

		h.reqCount.WithLabelValues(method, path).Inc()
		h.reqDuration.WithLabelValues(method, path, statusStr).Observe(duration)

		if rw.status >= 400 {
			h.errCount.WithLabelValues(method, path, statusStr).Inc()
		}
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (h *OrchestratorHandler) RegisterRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v1").Subrouter()

	api.HandleFunc("/jobs", h.withMetrics(h.handleCreateJob)).Methods(http.MethodPost)
	api.HandleFunc("/jobs", h.withMetrics(h.handleListJobs)).Methods(http.MethodGet)
	api.HandleFunc("/jobs/{id}", h.withMetrics(h.handleGetJob)).Methods(http.MethodGet)
	api.HandleFunc("/jobs/{id}", h.withMetrics(h.handleDeleteJob)).Methods(http.MethodDelete)
	api.HandleFunc("/jobs/{id}/files", h.withMetrics(h.handleGetFiles)).Methods(http.MethodGet)
	api.HandleFunc("/health", h.withMetrics(h.handleHealth)).Methods(http.MethodGet)
	api.HandleFunc("/jobs/{id}/deploy", h.withMetrics(h.handleDeploy)).Methods(http.MethodPost)

	// Prometheus
	r.Handle("/metrics", promhttp.Handler())
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

type createJobReq struct {
	Description string `json:"description"`
	Target      string `json:"target"`
}

// POST /api/v1/jobs
func (h *OrchestratorHandler) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req createJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("bad request body: %w", err))
		return
	}
	if req.Description == "" || req.Target == "" {
		writeError(w, http.StatusBadRequest, errors.New("description and target are required"))
		return
	}

	job := entity.NewJob(req.Description, req.Target)
	if err := h.jobService.CreateJob(r.Context(), req.Description, req.Target); err != nil {
		h.logger.Error("create job failed", "err", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, job)
}

// GET /api/v1/jobs
func (h *OrchestratorHandler) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.jobService.ListJobs(r.Context())
	if err != nil {
		h.logger.Error("list jobs failed", "err", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

// GET /api/v1/jobs/{id}
func (h *OrchestratorHandler) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("id required"))
		return
	}
	files, err := h.configFileService.GetFilesByJobID(r.Context(), id)
	if err != nil {
		h.logger.Error("get job failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if files == nil {
		writeError(w, http.StatusNotFound, errors.New("job not found"))
		return
	}
	writeJSON(w, http.StatusOK, files)
}

// DELETE /api/v1/jobs/{id}
func (h *OrchestratorHandler) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("id required"))
		return
	}
	if err := h.jobService.DeleteJob(r.Context(), id); err != nil {
		h.logger.Error("delete job failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// GET /api/v1/jobs/{id}/files
func (h *OrchestratorHandler) handleGetFiles(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("id required"))
		return
	}
	files, err := h.configFileService.GetFilesByJobID(r.Context(), id)
	if err != nil {
		h.logger.Error("get files failed", "job_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, files)
}

// POST /api/v1/jobs/{id}/deploy
func (h *OrchestratorHandler) handleDeploy(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	jobID := mux.Vars(r)["id"]

	if jobID == "" {
		http.Error(w, "job_id required", http.StatusBadRequest)
		return
	}

	if err := h.jobService.DeployJob(ctx, jobID); err != nil {
		if errors.Is(err, context.Canceled) {
			http.Error(w, "request canceled", http.StatusRequestTimeout)
			return
		}
		http.Error(w, "cannot deploy: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{"job_id": jobID, "status": "deploying"})
}

// GET /api/v1/health
func (h *OrchestratorHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"ok": true,
		"ts": time.Now().UTC(),
	}
	writeJSON(w, http.StatusOK, status)
}
