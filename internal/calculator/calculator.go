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

// CostCalculator chịu trách nhiệm tính toán
type CostCalculator struct {
	promAPI     prometheusAPI.API
	pricingConf *types.PricingConfig
}

// NewCostCalculator khởi tạo calculator
func NewCostCalculator(api prometheusAPI.API, pricing *types.PricingConfig) *CostCalculator {
	return &CostCalculator{
		promAPI:     api,
		pricingConf: pricing,
	}
}

// CalculatePodCosts là hàm chính thực hiện tính toán
func (cc *CostCalculator) CalculatePodCosts(ctx context.Context, start, end time.Time, step time.Duration) ([]types.PodCost, error) {
	if cc.pricingConf == nil {
		return nil, fmt.Errorf("pricing configuration is not loaded")
	}

	queryRange := prometheusAPI.Range{Start: start, End: end, Step: step}
	window := types.Window{Start: start, End: end}

	// --- 1. Truy vấn Prometheus song song ---
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

	wg.Wait() // Đợi tất cả query hoàn thành

	// Kiểm tra lỗi query (có thể làm chi tiết hơn)
	if cpuErr != nil {
		return nil, fmt.Errorf("error querying CPU usage: %w", cpuErr)
	}
	if ramErr != nil {
		return nil, fmt.Errorf("error querying RAM usage: %w", ramErr)
	}
	slog.Info("Prometheus queries completed.")

	// --- 2. Xử lý Kết quả Prometheus ---
	slog.Info("Parsing Prometheus results...")
	podCPUCoreSecondsMap := prom.ParseCPUUsage(cpuResult, step)
	podRAMByteSecondsMap := prom.ParseRAMUsage(ramResult, step)
	slog.Info("Parsing completed.")

	results := []types.PodCost{}

	// --- 3. Tính toán Chi phí cho từng Pod ---
	// Tạo danh sách pod key duy nhất từ cả CPU và RAM map
	allPodKeys := map[string]bool{}
	for key := range podCPUCoreSecondsMap {
		allPodKeys[key] = true
	}
	for key := range podRAMByteSecondsMap {
		allPodKeys[key] = true
	}

	slog.Info("Calculating costs", "unique_pods_found", len(allPodKeys))

	// *** Lấy giá mặc định MỘT LẦN ***
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

		// Lấy usage (mặc định là 0 nếu không có trong map)
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

		// // Tìm node
		// nodeName, nodeFound := podToNodeMap[podKey]
		// if !nodeFound {
		// 	errMsg := fmt.Sprintf("Node mapping not found via KSM (kube_pod_info)")
		// 	costEntry.Errors = append(costEntry.Errors, errMsg)
		// 	results = append(results, costEntry)
		// 	slog.Warn("Node mapping not found for pod", "pod_key", podKey)
		// 	continue
		// }

		// // Tìm labels của node
		// labels, labelsFound := nodeLabelsMap[nodeName]
		// if !labelsFound {
		// 	errMsg := fmt.Sprintf("Node labels not found via KSM (kube_node_labels) for node '%s'. Using default pricing.", nodeName)
		// 	costEntry.Errors = append(costEntry.Errors, errMsg)
		// 	slog.Warn("Node labels not found", "node", nodeName, "detail", "Using default pricing.")
		// }

		// // Xác định giá CPU và RAM
		// cpuPricePerHour := getCPUPriceForNode(cc.pricingConf, labels)
		// ramPricePerGiBHour := getRAMPriceForNode(cc.pricingConf, labels) // *** THÊM: Lấy giá RAM ***

		// Tính chi phí CPU
		costEntry.CPUCost = totalCPUCoreSeconds * (cpuPricePerHour / types.HoursToSeconds)

		// Tính chi phí RAM
		costEntry.RAMCost = totalRAMByteSeconds * (ramPricePerGiBHour / types.GiB / types.HoursToSeconds)

		// Tính TotalCost
		costEntry.TotalCost = costEntry.CPUCost + costEntry.RAMCost

		results = append(results, costEntry)
	}
	slog.Info("Calculation finished.", "pods_processed", len(results))

	return results, nil
}
