package main

import (
	"context"
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"payment-engine/internal/config"
	"payment-engine/internal/processor"
)

const (
	AccountAddressPrefix = "socone"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(AccountAddressPrefix, AccountAddressPrefix+"pub")

	config.Seal()
	log.Println("INFO: Bech32 prefixes configured for:", AccountAddressPrefix)
}

func main() {
	// --- Define Flags ---
	apiUrl := flag.String("api-url", "http://localhost:9991", "Base URL of the cost API server")
	apiWindow := flag.String("api-window", "15m", "Window parameter for the cost API (e.g., 5m, 15m, 1h)")
	apiStep := flag.String("api-step", "1m", "Step parameter for the cost API (e.g., 1m, 5m)")

	grpcAddress := flag.String("grpc-address", "localhost:9090", "gRPC endpoint of the streampayd node (host:port)")

	chainID := flag.String("chain-id", "sp-test-1", "StreamPay Chain ID (--chain-id)")
	providerAddress := flag.String("provider-address", "", "Address of the provider (REQUIRED)")
	stakeUnit := flag.String("stake-unit", "stake", "StreamPay currency (amount/fee suffix)")
	costToStakeRate := flag.Float64("rate", 1000.0, "Conversion rate from cost unit to stake unit (REQUIRED > 0)")
	minStakeAmount := flag.Int64("min-stake", 1, "Minimum stake amount to send payment (must be >= 1)")
	dryRun := flag.Bool("dry-run", false, "Run in simulation mode, do not execute deposit command")

	// --- Key Management Flags ---
	keyDirectory := flag.String("key-directory", "/keys", "Directory containing private key files (e.g., user1.pem)")

	// --- Gas Flags ---
	gasLimit := flag.Uint64("gas-limit", 200000, "Gas limit for transactions")
	gasFeeAmount := flag.Int64("gas-fee-amount", 10, "Amount for gas fee")
	gasFeeDenom := flag.String("gas-fee-denom", "stake", "Denomination for gas fee (use stake unit if empty)")

	interval := flag.Duration("interval", 15*time.Minute, "Frequency to run the payment cycle (e.g., 5m, 15m, 1h)")

	flag.Parse()

	// --- Validate Flags ---
	if *providerAddress == "" {
		log.Fatal("Error: Flag -provider-address is required.")
	}
	if *costToStakeRate <= 0 {
		log.Fatal("Error: Flag -rate must be positive.")
	}
	if *minStakeAmount < 1 {
		// Assume minimum unit is 1
		log.Fatal("Error: Flag -min-stake must be at least 1.")
	}
	// Ensure stakeUnit does not Empty and contain no spaces (simple)
	if *stakeUnit == "" || strings.Contains(*stakeUnit, " ") {
		log.Fatal("Error: Flag -stake-unit is invalid.")
	}

	// Validate apiUrl
	_, err := url.ParseRequestURI(*apiUrl)
	if err != nil {
		log.Fatalf("Error: Flag -api-url is not a valid URL: %v", err)
	}
	if *apiWindow == "" {
		log.Fatal("Error: Flag -api-window is required.")
	}
	if *apiStep == "" {
		log.Fatal("Error: Flag -api-step is required.")
	}

	if *grpcAddress == "" {
		log.Fatal("Error: Flag -grpc-address is required.")
	}

	// Basic check for key directory - Consider more robust checks
	info, err := os.Stat(*keyDirectory)
	if os.IsNotExist(err) {
		log.Fatalf("Error: Key directory specified by -key-directory does not exist: %s", *keyDirectory)
	} else if err != nil {
		log.Fatalf("Error checking key directory %s: %v", *keyDirectory, err)
	} else if !info.IsDir() {
		log.Fatalf("Error: Path specified by -key-directory is not a directory: %s", *keyDirectory)
	}

	// gas check
	if *gasLimit == 0 {
		log.Fatal("Error: Flag -gas-limit must be positive.")
	}
	if *gasFeeAmount < 0 {
		log.Fatal("Error: Flag -gas-fee-amount cannot be negative.")
	}
	actualGasFeeDenom := *gasFeeDenom
	if actualGasFeeDenom == "" {
		actualGasFeeDenom = *stakeUnit // Default gas fee denom to stake unit if not provided
	}
	if actualGasFeeDenom == "" {
		log.Fatal("Error: Gas fee denomination (-gas-fee-denom or -stake-unit) cannot be empty.")
	}

	if *interval <= 0 {
		log.Fatal("Error: Flag -interval must be a positive duration.")
	}

	// --- Create Config ---
	cfg := config.Config{
		ApiUrl:    *apiUrl,
		ApiWindow: *apiWindow,
		ApiStep:   *apiStep,

		GrpcAddress: *grpcAddress,

		KeyDirectory: *keyDirectory,

		ChainID:         *chainID,
		ProviderAddress: *providerAddress,
		StakeUnit:       *stakeUnit,
		CostToStakeRate: *costToStakeRate,
		MinStakeAmount:  *minStakeAmount,

		GasLimit:     *gasLimit,
		GasFeeAmount: *gasFeeAmount,
		GasFeeDenom:  actualGasFeeDenom,

		DryRun: *dryRun,
	}
	// --- Print loaded configuration ---
	log.Println("--- Payment Engine Configuration (gRPC Mode) ---")
	log.Printf(" API URL: %s", cfg.ApiUrl)
	log.Printf(" API Window: %s", cfg.ApiWindow)
	log.Printf(" API Step: %s", cfg.ApiStep)

	log.Printf(" gRPC Address: %s", cfg.GrpcAddress)
	log.Printf(" Key Directory: %s", cfg.KeyDirectory)

	log.Printf(" Chain ID: %s", cfg.ChainID)
	log.Printf(" Provider Address: %s", cfg.ProviderAddress)
	log.Printf(" Stake Unit: %s", cfg.StakeUnit)
	log.Printf(" Cost to Stake Rate: %.4f", cfg.CostToStakeRate)
	log.Printf(" Min Stake Amount: %d %s", cfg.MinStakeAmount, cfg.StakeUnit)
	log.Printf(" Gas Limit: %d", cfg.GasLimit)
	log.Printf(" Gas Fee: %d %s", cfg.GasFeeAmount, cfg.GasFeeDenom)
	log.Printf(" Dry Run Mode: %t", cfg.DryRun)
	log.Printf(" Interval: %s", interval.String()) // Log interval
	log.Println("----------------------------")

	// --- Setup Signal Handling for Graceful Shutdown ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Println("Running initial payment cycle...")
	runCycle(cfg) // Gọi hàm riêng để dễ đọc

	// --- Start Periodic Execution ---
	log.Printf("Starting periodic payment cycle every %s", interval.String())
	ticker := time.NewTicker(*interval)
	defer ticker.Stop() // Ensures ticker is stopped when main() finishes

	for {
		select {
		case <-ticker.C: // Waits for next tick
			log.Printf("Ticker triggered. Running payment cycle at %s...", time.Now().Format(time.RFC3339))
			runCycle(cfg)
		case <-ctx.Done(): // Receives stop signal (Ctrl+C)
			log.Println("Received shutdown signal. Stopping payment engine...")
			// Can perform final cleanup here if needed
			return // Exits the loop and terminates the program
		}
	}
}

// runCycle wraps the call to processor.RunPaymentCycle and handles logging/errors
func runCycle(cfg config.Config) {
	log.Println("----- Starting new payment cycle -----")
	err := processor.RunPaymentCycle(cfg) // This function needs to return an error for main to handle if desired
	if err != nil {
		// In periodic mode, do not exit(1) immediately when an error occurs
		// Just log the error and continue with the next cycle (unless the error is too serious)
		log.Printf("[ERROR] Payment cycle finished with errors: %v", err)
	} else {
		log.Println("----- Payment cycle finished successfully -----")
	}
}
