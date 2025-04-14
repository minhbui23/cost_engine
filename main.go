// /main.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"simple-cost-calculator/internal/calculator"
	"simple-cost-calculator/internal/config"
	"simple-cost-calculator/internal/prom"
	"simple-cost-calculator/internal/utils"
)

func main() {
	// --- Định nghĩa Flags ---
	promAddr := flag.String("prometheus.address", "http://localhost:9090", "Address of Prometheus server")
	pricingFile := flag.String("pricing.file", "configs/pricing.yaml", "Path to pricing configuration file (YAML)")
	windowStr := flag.String("window", "1h", "Calculation window duration (e.g., 5m, 1h, 24h)")
	stepStr := flag.String("step", "1m", "Calculation step duration (e.g., 1m, 5m, 15m)")
	debug := flag.Bool("debug", false, "Enable debug logging")

	webListenAddr := flag.String("web.listen-address", ":9991", "Address for the web server to listen on")
	webUiPath := flag.String("web.ui-path", "./ui", "Path to static UI files (HTML, CSS, JS)")
	webDataPath := flag.String("web.data-path", "./data", "Path to write and serve the costs.json file")

	flag.Parse()

	// --- Thiết lập Logger ---
	logger := utils.SetupLogger(*debug)
	slog.SetDefault(logger)

	// --- Xử lý Thời gian ---
	windowDuration, err := time.ParseDuration(*windowStr)
	if err != nil {
		logger.Error("Invalid window duration", "input", *windowStr, "error", err)
		os.Exit(1)
	}
	stepDuration, err := time.ParseDuration(*stepStr)
	if err != nil {
		logger.Error("Invalid step duration", "input", *stepStr, "error", err)
		os.Exit(1)
	}
	if stepDuration <= 0 {
		logger.Error("Step duration must be positive")
		os.Exit(1)
	}

	end := time.Now()
	start := end.Add(-windowDuration)

	logger.Info("Calculating costs", "window_start", start.Format(time.RFC3339), "window_end", end.Format(time.RFC3339), "step", stepDuration)

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
	calc := calculator.NewCostCalculator(promAPI, pricingConf /*, logger*/)
	logger.Info("Cost calculator initialized.")

	// --- Thực hiện Tính toán ---
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	podCosts, err := calc.CalculatePodCosts(ctx, start, end, stepDuration)
	if err != nil {
		logger.Error("Error calculating pod costs", "error", err)
	} else {
		logger.Info("Calculation complete.", "pods_found", len(podCosts))
	}

	// --- Chuẩn bị Thư mục và File Output JSON ---
	err = os.MkdirAll(*webDataPath, 0755) // Tạo thư mục data nếu chưa có
	if err != nil {
		logger.Error("Error creating data directory", "path", *webDataPath, "error", err)
		os.Exit(1)
	}

	jsonOutputFile := filepath.Join(*webDataPath, "costs.json")
	// Chỉ ghi file nếu tính toán thành công
	if err == nil {
		logger.Info("Writing calculation results to JSON file", "path", jsonOutputFile)
		outputFileHandle, err := os.Create(jsonOutputFile)
		if err != nil {
			logger.Error("Error creating output JSON file", "path", jsonOutputFile, "error", err)
			// Không exit, vẫn có thể chạy server với dữ liệu cũ (nếu có)
		} else {
			encoder := json.NewEncoder(outputFileHandle)
			encoder.SetIndent("", "  ") // Ghi JSON đẹp
			errEncode := encoder.Encode(podCosts)
			errClose := outputFileHandle.Close() // Đóng file ngay sau khi ghi xong
			if errEncode != nil {
				logger.Error("Error encoding results to JSON", "error", errEncode)
			}
			if errClose != nil {
				logger.Error("Error closing output JSON file", "error", errClose)
			}
		}
	} else {
		logger.Warn("Skipping writing JSON file due to calculation error. Server will start without updated data (or with old data if exists).")
	}

	//Rearrange costs by namespace
	result, err := calculator.RearrangeCosts(jsonOutputFile)
	if err != nil {
		logger.Error("Rearranging costs", "error", err)
	}

	userCostJsonOutputFile := filepath.Join(*webDataPath, "user_costs.json")

	outFile, err := os.Create(userCostJsonOutputFile)
	if err != nil {
		logger.Error("Error creating output JSON file", "path", userCostJsonOutputFile, "error", err)
	}
	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ") // format đẹp cho JSON

	if err := encoder.Encode(result); err != nil {
		slog.Error("Error to write JSON to file", "error", err)
	}

	logger.Info("Rearranged costs by namespace", "output_file", userCostJsonOutputFile)

	// --- Khởi chạy Web Server ---
	// Tạo một ServeMux mới để định tuyến
	mux := http.NewServeMux()

	// Phục vụ các file tĩnh trong thư mục UI (frontend)
	uiFileServer := http.FileServer(http.Dir(*webUiPath))
	mux.Handle("/", uiFileServer) // Phục vụ index.html và các file khác từ gốc

	// Phục vụ các file dữ liệu trong thư mục data (chứa costs.json) dưới prefix /data/
	datafileServer := http.FileServer(http.Dir(*webDataPath))
	mux.Handle("/data/", http.StripPrefix("/data/", datafileServer))

	logger.Info("Starting web server", "address", *webListenAddr, "ui_path", *webUiPath, "data_path", *webDataPath)

	// Khởi chạy server và block ở đây
	err = http.ListenAndServe(*webListenAddr, mux)
	if err != nil {
		logger.Error("Error starting web server", "error", err)
		os.Exit(1)
	}
}
