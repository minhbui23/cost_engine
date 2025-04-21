// internal/calculator/calculator.go

package calculator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"simple-cost-calculator/internal/prom"
	"simple-cost-calculator/internal/types"

	prometheusAPI "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type CostCalculator struct {
	promAPI     prometheusAPI.API
	pricingConf *types.PricingConfig
}

func NewCostCalculator(api prometheusAPI.API, pricing *types.PricingConfig) *CostCalculator {
	return &CostCalculator{
		promAPI:     api,
		pricingConf: pricing,
	}
}

// Main function to calculate costs for all pods in the given time range
func (cc *CostCalculator) CalculatePodCosts(ctx context.Context, start, end time.Time, step time.Duration) ([]types.PodCost, error) {
	if cc.pricingConf == nil {
		return nil, fmt.Errorf("pricing configuration is not loaded")
	}

	queryRange := prometheusAPI.Range{Start: start, End: end, Step: step}
	window := types.Window{Start: start, End: end}

	// --- 1. Query Prometheus for CPU and RAM usage ---
	slog.Info("Querying Prometheus (CPU, RAM, KSM)...")
	var wg sync.WaitGroup
	var cpuResult, ramResult model.Value
	var cpuErr, ramErr error

	wg.Add(2)

	//CPU Query
	go func() {
		defer wg.Done()
		cpuUsageQuery := fmt.Sprintf(prom.CPUUsageRateQueryTemplate, step.String())
		cpuResult, cpuErr = prom.QueryRange(ctx, cc.promAPI, cpuUsageQuery, queryRange)
	}()

	//RAM Query
	go func() {
		defer wg.Done()
		ramUsageQuery := fmt.Sprintf(prom.RAMUsageAvgBytesQueryTemplate, step.String())
		ramResult, ramErr = prom.QueryRange(ctx, cc.promAPI, ramUsageQuery, queryRange)
	}()

	wg.Wait()

	if cpuErr != nil {
		return nil, fmt.Errorf("error querying CPU usage: %w", cpuErr)
	}
	if ramErr != nil {
		return nil, fmt.Errorf("error querying RAM usage: %w", ramErr)
	}
	slog.Info("Prometheus queries completed.")

	// --- 2. Parse Prometheus results ---
	slog.Info("Parsing Prometheus results...")
	podCPUCoreSecondsMap := prom.ParseCPUUsage(cpuResult, step)
	podRAMByteSecondsMap := prom.ParseRAMUsage(ramResult, step)
	slog.Info("Parsing completed.")

	results := []types.PodCost{}

	allPodKeys := map[string]bool{}
	for key := range podCPUCoreSecondsMap {
		allPodKeys[key] = true
	}
	for key := range podRAMByteSecondsMap {
		allPodKeys[key] = true
	}

	slog.Info("Calculating costs", "unique_pods_found", len(allPodKeys))

	cpuPricePerHour := cc.pricingConf.DefaultCPUPricePerHour
	ramPricePerGiBHour := cc.pricingConf.DefaultRAMPricePerGBHour
	slog.Debug("Using default pricing", "cpu_per_core_hour", cpuPricePerHour, "ram_per_gib_hour", ramPricePerGiBHour)

	for podKey := range allPodKeys {
		parts := strings.SplitN(podKey, "/", 2)
		if len(parts) != 2 {
			slog.Warn("Skipping invalid pod key", "key", podKey)
			continue
		}
		namespace, podName := parts[0], parts[1]

		totalCPUCoreSeconds := podCPUCoreSecondsMap[podKey]
		totalRAMByteSeconds := podRAMByteSecondsMap[podKey]
		costEntry := types.PodCost{
			Namespace:    namespace,
			Pod:          podName,
			Window:       window,
			CPUCoreHours: totalCPUCoreSeconds / types.HoursToSeconds,
			RAMGiBHours:  totalRAMByteSeconds / types.GiB / types.HoursToSeconds,
			//Errors:       []string{},
		}

		costEntry.CPUCost = totalCPUCoreSeconds * (cpuPricePerHour / types.HoursToSeconds)

		costEntry.RAMCost = totalRAMByteSeconds * (ramPricePerGiBHour / types.GiB / types.HoursToSeconds)

		//TotalCost
		costEntry.TotalCost = costEntry.CPUCost + costEntry.RAMCost

		results = append(results, costEntry)
	}
	slog.Info("Calculation finished.", "pods_processed", len(results))

	return results, nil
}
