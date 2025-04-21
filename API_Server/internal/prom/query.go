// internal/prom/query.go

package prom

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	prometheusAPI "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	// Query to get CPU usage rate (core-seconds) per pod, step will be replaced
	CPUUsageRateQueryTemplate = `sum(rate(container_cpu_usage_seconds_total{image!="",container_label_io_kubernetes_pod_namespace!="",container_label_io_kubernetes_pod_name!=""}[%s])) by (container_label_io_kubernetes_pod_namespace, container_label_io_kubernetes_pod_name)`

	// Query to get RAM usage (bytes) per pod, step will be replaced
	RAMUsageAvgBytesQueryTemplate = `avg(avg_over_time(container_memory_working_set_bytes{image!="",container_label_io_kubernetes_pod_namespace!="",container_label_io_kubernetes_pod_name!="",container_label_io_cri_containerd_kind="container"}[%s])) by (container_label_io_kubernetes_pod_namespace, container_label_io_kubernetes_pod_name)`
)

// QueryRange performs a range query against Prometheus API and returns the result.
func QueryRange(ctx context.Context, api prometheusAPI.API, query string, queryRange prometheusAPI.Range) (model.Value, error) {
	result, warnings, err := api.QueryRange(ctx, query, queryRange)
	if err != nil {
		return nil, fmt.Errorf("prometheus range query failed for query '%s': %w", query, err)
	}
	if len(warnings) > 0 {
		slog.Warn("Prometheus query range warnings for query", "Query", query, "Warning", warnings)
	}
	return result, nil
}

// QueryInstant performs an instant query against Prometheus API and returns the result.
func QueryInstant(ctx context.Context, api prometheusAPI.API, query string, queryTime time.Time) (model.Value, error) {
	result, warnings, err := api.Query(ctx, query, queryTime)
	if err != nil {
		return nil, fmt.Errorf("prometheus instant query failed for query '%s': %w", query, err)
	}
	if len(warnings) > 0 {
		slog.Warn("Prometheus instant query warnings for query", "Query", query, "Warning", warnings)
	}
	return result, nil
}
