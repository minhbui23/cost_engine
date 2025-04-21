package calculator

import (
	"log/slog"
	"regexp"

	"simple-cost-calculator/internal/types"
)

func RearrangeCosts(podCosts []types.PodCost) (map[string]types.GroupedCostSummary, error) {
	if len(podCosts) == 0 {
		slog.Info("RearrangeCosts received empty podCosts slice, returning empty map.")
		return make(map[string]types.GroupedCostSummary), nil
	}

	intermediateResult := make(map[string]map[string]float64)
	windows := make(map[string]types.Window) // Save window for each group

	// Regex get namesapce type ns(anything)-user(digits)
	re := regexp.MustCompile(`^(?:ns.+)-(user\d+)$`)

	for _, pc := range podCosts {
		if pc.Namespace == "" {
			slog.Debug("Skipping pod cost entry with empty namespace during rearrange", "pod", pc.Pod)
			continue
		}
		groupKey := "system" // Default group key "system" if regex doesn't match
		originalNamespace := pc.Namespace

		matches := re.FindStringSubmatch(originalNamespace)
		if len(matches) == 2 {
			groupKey = matches[1]
		}

		if _, exists := intermediateResult[groupKey]; !exists {
			intermediateResult[groupKey] = make(map[string]float64)
			windows[groupKey] = pc.Window
		}

		intermediateResult[groupKey][originalNamespace] += pc.TotalCost
	}

	// make final result
	finalResult := make(map[string]types.GroupedCostSummary)

	for groupKey, namespaceCosts := range intermediateResult {
		summary := make(types.GroupedCostSummary)
		groupTotalCost := 0.0

		for ns, cost := range namespaceCosts {
			summary[ns] = cost
			groupTotalCost += cost
		}

		summary["totalCost"] = groupTotalCost
		summary["window"] = windows[groupKey]

		finalResult[groupKey] = summary
	}

	return finalResult, nil
}
