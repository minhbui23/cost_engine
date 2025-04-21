// internal/prom/client.go

package prom

import (
	"fmt"

	"github.com/prometheus/client_golang/api"
	prometheusAPI "github.com/prometheus/client_golang/api/prometheus/v1"
)

// NewPrometheusAPI creates a new Prometheus API client.
func NewPrometheusAPI(prometheusAddress string) (prometheusAPI.API, error) {
	client, err := api.NewClient(api.Config{Address: prometheusAddress})
	if err != nil {
		return nil, fmt.Errorf("error creating Prometheus client at %s: %w", prometheusAddress, err)
	}
	return prometheusAPI.NewAPI(client), nil
}
