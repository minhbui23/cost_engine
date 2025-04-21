// /main.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"simple-cost-calculator/internal/calculator"
	"simple-cost-calculator/internal/config"
	"simple-cost-calculator/internal/prom"
	"simple-cost-calculator/internal/utils"
)

var (
	calc        *calculator.CostCalculator
	logger      *slog.Logger
	defaultStep time.Duration
)

func main() {
	// --- Flags ---
	promAddr := flag.String("prometheus.address", "http://localhost:9090", "Address of Prometheus server")
	pricingFile := flag.String("pricing.file", "configs/pricing.yaml", "Path to pricing configuration file (YAML)")
	stepStr := flag.String("step", "1m", "Calculation step duration (e.g., 1m, 5m, 15m)")
	debug := flag.Bool("debug", false, "Enable debug logging")
	webListenAddr := flag.String("web.listen-address", ":9991", "Address for the web server to listen on")
	flag.Parse()

	// --- Setup Logger ---
	logger := utils.SetupLogger(*debug)
	slog.SetDefault(logger)

	stepDuration, err := time.ParseDuration(*stepStr)
	if err != nil {
		logger.Error("Invalid step duration", "input", *stepStr, "error", err)
		os.Exit(1)
	}
	if stepDuration <= 0 {
		logger.Error("Step duration must be positive")
		os.Exit(1)
	}
	defaultStep = stepDuration
	// --- Load Pricing Config ---
	logger.Info("Loading pricing config", "path", *pricingFile)
	pricingConf, err := config.LoadPricingConfig(*pricingFile)
	if err != nil {
		logger.Error("Error loading pricing config", "error", err)
		os.Exit(1)
	}
	logger.Info("Pricing config loaded successfully.")

	// --- Initit Prometheus API Client ---
	logger.Info("Connecting to Prometheus", "address", *promAddr)
	promAPI, err := prom.NewPrometheusAPI(*promAddr)

	if err != nil {
		logger.Error("Error creating Prometheus client", "error", err)
		os.Exit(1)
	}
	logger.Info("Prometheus client created.")

	// --- Cost Calculator ---
	calc = calculator.NewCostCalculator(promAPI, pricingConf /*, logger*/)
	logger.Info("Cost calculator initialized.")

	// --- Web Server ---
	mux := http.NewServeMux()

	mux.HandleFunc("/getcost", handleGetCost)

	slog.Info("Starting API server with ", "address", *webListenAddr)

	err = http.ListenAndServe(*webListenAddr, mux)
	if err != nil {
		slog.Error("Error starting API server", "error", err)
		os.Exit(1)
	}
}

func handleGetCost(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	windowQuery := r.URL.Query().Get("window")
	if windowQuery == "" {
		slog.Warn("API request missing 'window' parameter")
		http.Error(w, "Missing 'window' query parameter (e.g., ?window=5m)", http.StatusBadRequest)
		return
	}

	windowDuration, err := time.ParseDuration(windowQuery)
	if err != nil {
		slog.Warn("API request invalid 'window' format", "input", windowQuery, "error", err) // Dùng slog.Warn
		http.Error(w, fmt.Sprintf("Invalid 'window' duration format: %v. Use format like '5m', '1h'.", err), http.StatusBadRequest)
		return
	}

	stepQuery := r.URL.Query().Get("step")
	step := defaultStep
	if stepQuery != "" {
		stepDurationQuery, err := time.ParseDuration(stepQuery)
		if err == nil && stepDurationQuery > 0 {
			step = stepDurationQuery
		} else {
			slog.Warn("API request invalid 'step' format, using default", "input", stepQuery, "default", defaultStep, "error", err)
			http.Error(w, fmt.Sprintf("Invalid 'step' duration format: %v", err), http.StatusBadRequest)
			return
		}
	}

	end := time.Now()
	start := end.Add(-windowDuration)

	slog.Info("API request received", "window", windowDuration, "step", step, "start", start.Format(time.RFC3339), "end", end.Format(time.RFC3339))

	podCosts, err := calc.CalculatePodCosts(ctx, start, end, step)
	if err != nil {
		slog.Error("Error calculating pod costs via API", "window", windowDuration, "step", step, "error", err)
		http.Error(w, "Internal Server Error: Failed to calculate costs.", http.StatusInternalServerError)
		return
	}

	if len(podCosts) == 0 {
		slog.Info("No pod cost data found for the requested window via API", "window", windowDuration, "step", step)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "{}")
		return
	}

	slog.Info("Pod costs calculated successfully via API", "pod_count", len(podCosts))

	rearrangedCosts, err := calculator.RearrangeCosts(podCosts)
	if err != nil {
		slog.Error("Error rearranging costs via API", "error", err)
		http.Error(w, "Internal Server Error: Failed to process results.", http.StatusInternalServerError)
		return
	}

	slog.Info("Costs rearranged successfully via API", "user_groups", len(rearrangedCosts))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	errEncode := json.NewEncoder(w).Encode(rearrangedCosts)
	if errEncode != nil {
		slog.Error("Error encoding JSON response", "error", errEncode)
	}
}
