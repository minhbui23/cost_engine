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

	"github.com/rs/cors"
)

var (
	calc        *calculator.CostCalculator
	logger      *slog.Logger  // Logger toàn cục
	defaultStep time.Duration // Lấy từ flag
)

func main() {
	// --- Định nghĩa Flags ---
	promAddr := flag.String("prometheus.address", "http://localhost:9090", "Address of Prometheus server")
	pricingFile := flag.String("pricing.file", "configs/pricing.yaml", "Path to pricing configuration file (YAML)")
	stepStr := flag.String("step", "1m", "Calculation step duration (e.g., 1m, 5m, 15m)")
	debug := flag.Bool("debug", false, "Enable debug logging")
	webListenAddr := flag.String("web.listen-address", ":9991", "Address for the web server to listen on")
	uiPort := flag.String("ui.port", "8080", "Port for the UI to connect to")
	flag.Parse()

	// --- Thiết lập Logger ---
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

	// --- Khởi tạo Prometheus API Client ---
	logger.Info("Connecting to Prometheus", "address", *promAddr)
	promAPI, err := prom.NewPrometheusAPI(*promAddr)

	if err != nil {
		logger.Error("Error creating Prometheus client", "error", err)
		os.Exit(1)
	}
	logger.Info("Prometheus client created.")

	// --- Khởi tạo Cost Calculator ---
	calc = calculator.NewCostCalculator(promAPI, pricingConf /*, logger*/)
	logger.Info("Cost calculator initialized.")

	// --- Khởi tạo Web Server ---
	mux := http.NewServeMux()

	mux.HandleFunc("/getcost", handleGetCost)

	uiOriginLocalhost := "http://localhost:" + *uiPort // Tạo Origin dựa trên flag hoặc hardcode
	uiOriginIP := "http://192.168.10.130:" + *uiPort
	allowedOrigins := []string{uiOriginLocalhost, uiOriginIP}
	// Hoặc cho phép tất cả cho test local (kém an toàn hơn):
	// allowedOrigins = []string{"*"}

	// Cấu hình chi tiết CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,                                         // Chỉ cho phép nguồn từ UI
		AllowedMethods:   []string{http.MethodGet, http.MethodOptions},           // Cho phép method GET và OPTIONS (preflight)
		AllowedHeaders:   []string{"Accept", "Content-Type", "X-Requested-With"}, // Cho phép các header thông thường
		AllowCredentials: true,                                                   // Có thể cần nếu có xác thực sau này
		MaxAge:           300,                                                    // Cache preflight request trong 5 phút
		Debug:            *debug,                                                 // Bật debug CORS nếu cần
	})

	// Bọc mux bằng CORS middleware
	handler := c.Handler(mux)

	slog.Info("Starting API server with CORS", "address", *webListenAddr)

	// Chạy server với handler đã bọc CORS
	err = http.ListenAndServe(*webListenAddr, handler)
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
		slog.Warn("API request missing 'window' parameter") // Dùng slog.Warn
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
	step := defaultStep // Dùng global defaultStep
	if stepQuery != "" {
		stepDurationQuery, err := time.ParseDuration(stepQuery)
		if err == nil && stepDurationQuery > 0 {
			step = stepDurationQuery
		} else {
			// Kiểm tra lỗi parse step
			slog.Warn("API request invalid 'step' format, using default", "input", stepQuery, "default", defaultStep, "error", err) // Dùng slog.Warn
			http.Error(w, fmt.Sprintf("Invalid 'step' duration format: %v", err), http.StatusBadRequest)
			return // Thêm return ở đây để dừng nếu step không hợp lệ
		}
	}

	end := time.Now()
	start := end.Add(-windowDuration)

	slog.Info("API request received", "window", windowDuration, "step", step, "start", start.Format(time.RFC3339), "end", end.Format(time.RFC3339)) // Dùng slog.Info

	// Gọi calc (biến toàn cục)
	podCosts, err := calc.CalculatePodCosts(ctx, start, end, step)
	if err != nil {
		slog.Error("Error calculating pod costs via API", "window", windowDuration, "step", step, "error", err) // Dùng slog.Error
		http.Error(w, "Internal Server Error: Failed to calculate costs.", http.StatusInternalServerError)
		return
	}

	if len(podCosts) == 0 {
		slog.Info("No pod cost data found for the requested window via API", "window", windowDuration, "step", step) // Dùng slog.Info
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "{}")
		return
	}

	slog.Info("Pod costs calculated successfully via API", "pod_count", len(podCosts)) // Dùng slog.Info

	rearrangedCosts, err := calculator.RearrangeCosts(podCosts)
	if err != nil {
		slog.Error("Error rearranging costs via API", "error", err) // Dùng slog.Error
		http.Error(w, "Internal Server Error: Failed to process results.", http.StatusInternalServerError)
		return
	}

	slog.Info("Costs rearranged successfully via API", "user_groups", len(rearrangedCosts)) // Dùng slog.Info

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	errEncode := json.NewEncoder(w).Encode(rearrangedCosts) // Đổi tên biến err tránh trùng lặp
	if errEncode != nil {
		slog.Error("Error encoding JSON response", "error", errEncode) // Dùng slog.Error
	}
}
