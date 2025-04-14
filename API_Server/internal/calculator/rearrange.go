package calculator

import (
	"log/slog"
	"regexp"

	"simple-cost-calculator/internal/types"
)

func RearrangeCosts(podCosts []types.PodCost) (map[string]types.GroupedCostSummary, error) {
	// Đọc file JSON
	if len(podCosts) == 0 {
		slog.Info("RearrangeCosts received empty podCosts slice, returning empty map.")
		return make(map[string]types.GroupedCostSummary), nil // Trả về map rỗng, không lỗi
	}

	// Sử dụng map trung gian để dễ tính toán tổng hợp
	intermediateResult := make(map[string]map[string]float64)
	windows := make(map[string]types.Window) // Lưu trữ window cho mỗi group

	// Regex để xác định namespace có dạng ns(anything)-us(digits)
	// Nó sẽ bắt nhóm (us\d+)
	re := regexp.MustCompile(`^(?:ns.+)-(user\d+)$`)

	for _, pc := range podCosts {
		if pc.Namespace == "" {
			slog.Debug("Skipping pod cost entry with empty namespace during rearrange", "pod", pc.Pod)
			continue
		}
		groupKey := "system" // Mặc định là system
		originalNamespace := pc.Namespace

		matches := re.FindStringSubmatch(originalNamespace)
		if len(matches) == 2 {
			// Nếu khớp regex, groupKey là phần "usX" (matches[1])
			groupKey = matches[1]
		}

		// Khởi tạo map cho group nếu chưa tồn tại
		if _, exists := intermediateResult[groupKey]; !exists {
			intermediateResult[groupKey] = make(map[string]float64)
			// Lưu window đầu tiên gặp cho group này (giả định chúng giống nhau)
			windows[groupKey] = pc.Window
		}

		// Cộng dồn totalCost cho namespace gốc trong group
		intermediateResult[groupKey][originalNamespace] += pc.TotalCost
	}

	// Tạo cấu trúc output cuối cùng theo yêu cầu
	finalResult := make(map[string]types.GroupedCostSummary)

	for groupKey, namespaceCosts := range intermediateResult {
		summary := make(types.GroupedCostSummary)
		groupTotalCost := 0.0

		// Thêm chi phí của từng namespace gốc vào summary
		for ns, cost := range namespaceCosts {
			summary[ns] = cost
			groupTotalCost += cost // Tính tổng chi phí cho group
		}

		// Thêm tổng chi phí và window vào summary
		summary["totalCost"] = groupTotalCost
		summary["window"] = windows[groupKey] // Lấy window đã lưu

		finalResult[groupKey] = summary
	}

	return finalResult, nil
}
