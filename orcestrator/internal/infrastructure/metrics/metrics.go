package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Jobs
	JobsCreated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "llmgen_jobs_created_total",
			Help: "Total number of jobs created",
		},
	)
	JobStatusChanges = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmgen_job_status_changes_total",
			Help: "Number of job status transitions",
		},
		[]string{"from", "to"},
	)
	ActiveJobs = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "llmgen_jobs_active",
			Help: "Current number of active jobs",
		},
	)
	JobDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "llmgen_job_duration_seconds",
			Help:    "Histogram of job durations in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 8), // 1s..128s
		},
	)

	// Validation
	ValidationRuns = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmgen_validation_runs_total",
			Help: "Number of validation runs by validator type and result",
		},
		[]string{"validator", "result"}, // validator: static|sandbox|security|codechecker, result: pass|fail|error
	)
	ValidationDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llmgen_validation_duration_seconds",
			Help:    "Duration of validation runs",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"validator"},
	)

	// Deployer / deploy flow
	DeployRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "llmgen_deploy_requests_total",
			Help: "Total number of deploy requests (plan/apply) initiated",
		},
	)
	DeployConfirms = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmgen_deploy_confirmations_total",
			Help: "Deploy confirmations by result",
		},
		[]string{"result"}, // result: confirmed|rejected
	)

	// LLM & code actions
	LLMRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmgen_llm_requests_total",
			Help: "Number of LLM requests by model/type",
		},
		[]string{"model"},
	)

	// DB / file storage ops
	DBFileOps = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmgen_db_file_ops_total",
			Help: "Database file operations performed",
		},
		[]string{"op"}, // op: get|put|delete|list
	)

	// Websockets / realtime
	// WebsocketConnections = prometheus.NewGauge(
	// 	prometheus.GaugeOpts{
	// 		Name: "llmgen_ws_connections",
	// 		Help: "Current number of open websocket connections",
	// 	},
	// )

	// Errors
	Errors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmgen_errors_total",
			Help: "Errors encountered in components",
		},
		[]string{"component", "type"},
	)
)

func init() {
	prometheus.MustRegister(
		// Jobs
		JobsCreated,
		JobStatusChanges,
		ActiveJobs,

		// Validation
		ValidationRuns,
		ValidationDurationSeconds,
		// Deploy
		DeployRequests,
		DeployConfirms,
		// LLM / code
		LLMRequests,

		// DB
		DBFileOps,
		// WS
		// WebsocketConnections,
		// Errors
		Errors,
	)
}

func StartMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(":2112", nil)
}

// Jobs
func IncJobsCreated() {
	JobsCreated.Inc()
}

func IncJobStatusChange(from, to string) {
	JobStatusChanges.WithLabelValues(from, to).Inc()
}

func SetActiveJobs(n int) {
	ActiveJobs.Set(float64(n))
}

func ObserveJobDuration(d time.Duration) {
	JobDurationSeconds.Observe(d.Seconds())
}

// Validation
func IncValidationRun(validator, result string) {
	ValidationRuns.WithLabelValues(validator, result).Inc()
}

func ObserveValidationDuration(validator string, d time.Duration) {
	ValidationDurationSeconds.WithLabelValues(validator).Observe(d.Seconds())
}

// Deployer
func IncDeployRequest() {
	DeployRequests.Inc()
}

func IncDeployConfirm(result string) {
	DeployConfirms.WithLabelValues(result).Inc()
}

// LLM
func IncLLMRequest(model string) {
	LLMRequests.WithLabelValues(model).Inc()
}

// DB / file ops
func IncDBFileOp(op string) {
	DBFileOps.WithLabelValues(op).Inc()
}

// Websocket
// func IncWSConnections() {
// 	WebsocketConnections.Inc()
// }

// func DecWSConnections() {
// 	WebsocketConnections.Dec()
// }

// func SetWSConnections(n int) {
// 	WebsocketConnections.Set(float64(n))
// }

// Errors
func IncError(component, typ string) {
	Errors.WithLabelValues(component, typ).Inc()
}
