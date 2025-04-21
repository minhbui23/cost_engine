// types/types.go
package types

import "time"

// PricingConfig define pricing configuration for CPU and RAM
type PricingConfig struct {
	DefaultCPUPricePerHour   float64            `yaml:"defaultCPUPricePerHour"`
	CPUPriceByInstanceType   map[string]float64 `yaml:"cpuPriceByInstanceType"`
	DefaultRAMPricePerGBHour float64            `yaml:"defaultRAMPricePerGBHour"`
	RAMPriceByInstanceType   map[string]float64 `yaml:"ramPriceByInstanceType"`
	// Add GPU and other resources if needed
}

// PodCPUCost define cost for a pod
type PodCost struct {
	Namespace    string  `json:"namespace"`
	Pod          string  `json:"pod"`
	Window       Window  `json:"window"`
	CPUCost      float64 `json:"cpuCost"`
	CPUCoreHours float64 `json:"cpuCoreHours"`

	RAMCost     float64 `json:"ramCost"`
	RAMGiBHours float64 `json:"ramGiBHours"`

	TotalCost float64 `json:"totalCost"`
	// Errors    []string `json:"errors,omitempty"`
}

type GroupedCostSummary map[string]interface{}

// Window time window for cost calculation
type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

const GiB = 1024 * 1024 * 1024
const HoursToSeconds = 3600.0
