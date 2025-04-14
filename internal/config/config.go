// internal/config/config.go

package config

import (
	"fmt"
	"os"

	"simple-cost-calculator/internal/types" // Import từ package types

	"gopkg.in/yaml.v3"
)

// LoadPricingConfig đọc và phân tích file cấu hình giá YAML
func LoadPricingConfig(filePath string) (*types.PricingConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading pricing file '%s': %w", filePath, err)
	}

	var config types.PricingConfig
	// Đặt giá trị mặc định trước khi unmarshal
	config.CPUPriceByInstanceType = make(map[string]float64) // Khởi tạo map
	config.RAMPriceByInstanceType = make(map[string]float64)

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling pricing config '%s': %w", filePath, err)
	}

	// Validation
	if config.DefaultCPUPricePerHour <= 0 {
		return nil, fmt.Errorf("invalid defaultCPUPricePerHour (<= 0) in pricing config '%s'", filePath)
	}

	if config.DefaultRAMPricePerGBHour <= 0 {
		return nil, fmt.Errorf("invalid defaultRAMPricePerGBHour (<= 0) in pricing config '%s'", filePath)
	}

	return &config, nil
}
