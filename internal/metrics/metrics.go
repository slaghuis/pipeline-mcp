package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Runs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_mcp_runs_total",
			Help: "Pipeline runs by service, command, and outcome.",
		},
		[]string{"service", "command", "status"},
	)

	Duration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pipeline_mcp_run_duration_seconds",
			Help:    "Pipeline run durations.",
			Buckets: []float64{10, 30, 60, 120, 300, 600, 1800},
		},
		[]string{"service", "command"},
	)

	StageDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pipeline_mcp_stage_duration_seconds",
			Help:    "Individual stage durations.",
			Buckets: []float64{1, 5, 15, 30, 60, 180, 600},
		},
		[]string{"service", "stage"},
	)

	StageFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pipeline_mcp_stage_failures_total",
			Help: "Stage failures by service and stage.",
		},
		[]string{"service", "stage"},
	)
)

func init() {
	prometheus.MustRegister(Runs, Duration, StageDuration, StageFailures)
}

func Handler() http.Handler { return promhttp.Handler() }