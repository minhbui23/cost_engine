// internal/prom/parser.go

package prom

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/prometheus/common/model"
)

func GetPodKey(namespace, pod string) string {
	return fmt.Sprintf("%s/%s", namespace, pod)
}

// ParseCPUUsage query result CPU to map[namespace/pod] -> totalCoreSeconds
func ParseCPUUsage(result model.Value, step time.Duration) map[string]float64 {
	slog.Debug("Entering ParseCPUUsage")
	podUsage := make(map[string]float64)
	matrix, ok := result.(model.Matrix)
	if !ok {
		slog.Warn(
			"ParseCPUUsage expected matrix type",
			"expected", "model.Matrix",
			"received", fmt.Sprintf("%T", result),
		)
		return podUsage
	}

	slog.Debug("ParseCPUUsage processing matrix", "series_count", len(matrix))

	for i, sampleStream := range matrix {
		metric := sampleStream.Metric

		namespace := string(metric["container_label_io_kubernetes_pod_namespace"])
		pod := string(metric["container_label_io_kubernetes_pod_name"])

		slog.Debug("Processing CPU series", "series_index", i, "raw_labels", metric)

		if namespace == "" || pod == "" {
			slog.Debug(
				"Skipping CPU series",
				"series_index", i,
				"reason", "missing k8s labels",
				"namespace", namespace,
				"pod", pod,
				"namespace_label_key", "container_label_io_kubernetes_pod_namespace",
				"pod_label_key", "container_label_io_kubernetes_pod_name",
			)
			continue
		}
		podKey := GetPodKey(namespace, pod)

		slog.Debug("Processing CPU series for pod", "series_index", i, "pod_key", podKey)

		var totalCoreSecs float64
		var pointsProcessed int
		for _, pair := range sampleStream.Values {
			value, err := strconv.ParseFloat(pair.Value.String(), 64)
			if err == nil && !isNaN(value) {
				stepSeconds := step.Seconds()
				pointCoreSecs := value * stepSeconds
				totalCoreSecs += pointCoreSecs
				pointsProcessed++
			} else if err != nil {
				slog.Warn(
					"Could not parse CPU value",
					"raw_value", pair.Value.String(),
					"pod_key", podKey,
					"timestamp", pair.Timestamp.Time(),
					"error", err,
				)
			} else if isNaN(value) {
				slog.Debug(
					"Skipping NaN CPU value",
					"pod_key", podKey,
					"timestamp", pair.Timestamp.Time(),
				)
			}
		}
		podUsage[podKey] += totalCoreSecs
		slog.Debug(
			"Finished processing CPU series for pod",
			"series_index", i,
			"pod_key", podKey,
			"points_processed", pointsProcessed,
			"total_core_seconds", totalCoreSecs,
		)
	}

	slog.Debug("Exiting ParseCPUUsage", "final_map_size", len(podUsage))
	return podUsage
}

// ParseRAMUsage query result RAM to map[namespace/pod] -> totalByteSeconds
func ParseRAMUsage(result model.Value, step time.Duration) map[string]float64 {

	slog.Debug("Entering ParseRAMUsage")
	podUsage := make(map[string]float64)
	matrix, ok := result.(model.Matrix)
	if !ok {
		slog.Warn(
			"ParseRAMUsage expected matrix type",
			"expected", "model.Matrix",
			"received", fmt.Sprintf("%T", result),
		)
		return podUsage
	}

	slog.Debug("ParseRAMUsage processing matrix", "series_count", len(matrix))

	for i, sampleStream := range matrix {
		metric := sampleStream.Metric
		namespace := string(metric["container_label_io_kubernetes_pod_namespace"])
		pod := string(metric["container_label_io_kubernetes_pod_name"])

		slog.Debug("Processing RAM series", "series_index", i, "raw_labels", metric)

		if namespace == "" || pod == "" {
			slog.Debug(
				"Skipping RAM series",
				"series_index", i,
				"reason", "missing k8s labels",
				"namespace", namespace,
				"pod", pod,
				"namespace_label_key", "container_label_io_kubernetes_pod_namespace",
				"pod_label_key", "container_label_io_kubernetes_pod_name",
			)
			continue
		}
		podKey := GetPodKey(namespace, pod)

		slog.Debug("Processing RAM series for pod", "series_index", i, "pod_key", podKey)

		var totalByteSecs float64
		var pointsProcessed int
		for _, pair := range sampleStream.Values {
			avgBytes, err := strconv.ParseFloat(pair.Value.String(), 64)
			if err == nil && !isNaN(avgBytes) {
				stepSeconds := step.Seconds()
				pointByteSecs := avgBytes * stepSeconds
				totalByteSecs += pointByteSecs
				pointsProcessed++
			} else if err != nil {
				slog.Warn(
					"Could not parse RAM value",
					"raw_value", pair.Value.String(),
					"pod_key", podKey,
					"timestamp", pair.Timestamp.Time(),
					"error", err,
				)
			} else if isNaN(avgBytes) {
				slog.Debug(
					"Skipping NaN RAM value",
					"pod_key", podKey,
					"timestamp", pair.Timestamp.Time(),
				)
			}
		}
		podUsage[podKey] += totalByteSecs
		slog.Debug(
			"Finished processing RAM series for pod",
			"series_index", i,
			"pod_key", podKey,
			"points_processed", pointsProcessed,
			"total_byte_seconds", totalByteSecs,
		)
	}

	slog.Debug("Exiting ParseRAMUsage", "final_map_size", len(podUsage))
	return podUsage
}

func isNaN(f float64) bool {
	return f != f
}
