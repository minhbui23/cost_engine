package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"payment-engine/internal/config"
	"payment-engine/internal/processor"
)

func main() {
	// --- Define Flags ---
	// Use Ptr to easily check if flags are set if needed
	costFile := flag.String("cost-file", "costs.json", "Path to JSON file containing cost data")
	streampaydPath := flag.String("streampayd-path", "streampayd", "Path or command name to execute streampayd")
	chainID := flag.String("chain-id", "sp-test-1", "StreamPay Chain ID (--chain-id)")
	providerAddress := flag.String("provider-address", "", "Address of the provider (REQUIRED)")
	keyringBackend := flag.String("keyring-backend", "test", "Keyring backend (--keyring-backend)")
	streamDuration := flag.String("stream-duration", "5m", "Stream time (--duration for stream-send)")
	stakeUnit := flag.String("stake-unit", "stake", "StreamPay currency (amount/fee suffix)")
	costToStakeRate := flag.Float64("rate", 1000.0, "Conversion rate from cost unit to stake unit (REQUIRED > 0)")
	minStakeAmount := flag.Int64("min-stake", 1, "Minimum stake amount to send payment (must be >= 1)")
	dryRun := flag.Bool("dry-run", false, "Run in simulation mode, do not execute deposit command")

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

	// --- Create Config ---
	cfg := config.Config{
		CostFile:        *costFile,
		StreampaydPath:  *streampaydPath,
		ChainID:         *chainID,
		ProviderAddress: *providerAddress,
		KeyringBackend:  *keyringBackend,
		StreamDuration:  *streamDuration,
		StakeUnit:       *stakeUnit,
		CostToStakeRate: *costToStakeRate,
		MinStakeAmount:  *minStakeAmount,
		DryRun:          *dryRun,
	}
	// --- Print loaded configuration ---
	log.Println("--- Payment Engine Configuration ---")
	log.Printf(" File Cost: %s", cfg.CostFile)
	log.Printf(" Streampayd Path: %s", cfg.StreampaydPath)
	log.Printf("Chain ID: %s", cfg.ChainID)
	log.Printf("Provider Address: %s", cfg.ProviderAddress)
	log.Printf(" Keyring Backend: %s", cfg.KeyringBackend)
	log.Printf(" Stream Duration: %s", cfg.StreamDuration)
	log.Printf(" Stake Unit: %s", cfg.StakeUnit)
	log.Printf(" Cost to Stake Rate: %.4f", cfg.CostToStakeRate)
	log.Printf(" Min Stake Amount: %d %s", cfg.MinStakeAmount, cfg.StakeUnit)
	log.Printf(" Dry Run Mode: %t", cfg.DryRun)
	log.Println("----------------------------")

	log.Println("Starting single payment processing cycle...")
	err := processor.RunPaymentCycle(cfg)
	if err != nil {
		log.Printf("[ERROR] Payment cycle finished with errors: %v", err)
		os.Exit(1)
	} else {
		log.Println("Payment cycle finished successfully.")
		os.Exit(0) // exit with success
	}
}
