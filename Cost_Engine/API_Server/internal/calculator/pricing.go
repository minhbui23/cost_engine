// // internal/calculator/pricing.go

package calculator

// // Các key label phổ biến cho instance type
// var instanceTypeLabelKeys = []string{
// 	"node.kubernetes.io/instance-type", // KSM > v1.6
// 	"beta.kubernetes.io/instance-type", // Label cũ hơn
// 	// Thêm các label key bạn sử dụng ở đây nếu khác
// 	"custom-node-type",
// }

// // getCPUPriceForNode xác định giá CPU dựa trên labels của node và config
// // Trả về giá mỗi core mỗi giờ
// func getCPUPriceForNode(pricingConf *types.PricingConfig, nodeLabels map[string]string) float64 {
// 	if pricingConf == nil {
// 		slog.Error("Error: Pricing config is nil in getCPUPriceForNode")
// 		return 0.0 // Hoặc một giá trị mặc định an toàn
// 	}

// 	if nodeLabels != nil {
// 		for _, key := range instanceTypeLabelKeys {
// 			if instanceType, ok := nodeLabels[key]; ok {
// 				if price, exists := pricingConf.CPUPriceByInstanceType[instanceType]; exists {
// 					slog.Debug("Debug: Found CPU price for instance type", "Label key", key, "Instance type", instanceType, "Price per hour", price)
// 					return price
// 				}
// 			}
// 		}
// 	}

// 	// Không tìm thấy giá cụ thể, sử dụng giá mặc định
// 	slog.Debug("Debug: Using default CPU price.", "CPU Price", pricingConf.DefaultCPUPricePerHour)
// 	return pricingConf.DefaultCPUPricePerHour
// }

// // getRAMPriceForNode xác định giá RAM dựa trên labels của node và config
// // Trả về giá mỗi GiB mỗi giờ ($/GiB-hour)
// func getRAMPriceForNode(pricingConf *types.PricingConfig, nodeLabels map[string]string) float64 {
// 	if pricingConf == nil {
// 		slog.Error("Error: Pricing config is nil in getRAMPriceForNode")
// 		return 0.0
// 	}

// 	if nodeLabels != nil {
// 		for _, key := range instanceTypeLabelKeys { // Dùng chung key label với CPU
// 			if instanceType, ok := nodeLabels[key]; ok {
// 				// Tìm trong map giá RAM theo instance type
// 				if price, exists := pricingConf.RAMPriceByInstanceType[instanceType]; exists {
// 					slog.Debug("Debug: Found RAM price by instance type", "Label key", key, "Instance type", instanceType, "Price per hour", price)
// 					return price
// 				}
// 			}
// 		}
// 	}

// 	// Không tìm thấy giá cụ thể, sử dụng giá mặc định
// 	slog.Debug("Debug: Using default RAM price", "RAM Price", pricingConf.DefaultRAMPricePerGBHour)
// 	return pricingConf.DefaultRAMPricePerGBHour
// }
