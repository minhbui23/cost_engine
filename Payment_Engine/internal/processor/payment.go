package processor

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"payment-engine/internal/config"
	"payment-engine/internal/parser"
	"payment-engine/internal/streampay"
)

const (
	feePercentage = 0.01 // 1% fee
	minFeeAmount  = 1
)

// RunPaymentCycle performs a complete payment processing cycle.
func RunPaymentCycle(cfg config.Config) error {
	log.Printf("===== Start payment cycle at %s =====", time.Now().Format(time.RFC3339))
	if cfg.DryRun {
		log.Println("[DRY RUN MODE ENABLED] Will not execute deposit command.")
	}

	// 1. Read and Parse cost file
	log.Printf("Reading cost file: %s", cfg.CostFile)
	costData, err := parser.ParseCostFile(cfg.CostFile)
	if err != nil {
		// If the error is due to the file not existing, consider it as nothing to process, not a serious error
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Cost file '%s' does not exist. Skip cycle this.", cfg.CostFile)
			log.Printf("===== End of cycle (no file) at %s =====", time.Now().Format(time.RFC3339))
			return nil // Not an error, just nothing to do
		}
		// Other errors (parse JSON, read file) are errors to report
		log.Printf("[FATAL ERROR] Unable to read or parse file cost '%s': %v", cfg.CostFile, err)
		log.Printf("===== End of cycle (error reading file) at %s =====", time.Now().Format(time.RFC3339))
		return fmt.Errorf("error processing file cost: %w", err) // Returns an error so main can handle it
	}

	if len(costData) == 0 {
		log.Println("The cost file is empty or does not contain valid data. End of cycle.")
		log.Printf("===== End of cycle (empty file) at %s =====", time.Now().Format(time.RFC3339))
		return nil
	}
	log.Printf("Successfully read and parsed %d items from cost file.", len(costData))

	// 2. Loop through each user and process
	successCount := 0
	skippedCount := 0
	addressErrorCount := 0
	sendErrorCount := 0

	for userID, userData := range costData {
		// Skip system users
		if strings.ToLower(userID) == "system" {
			// log.Printf("Skip system users: %s", userID)
			continue
		}

		log.Printf("--- Processing User: %s ---", userID)
		log.Printf(" Original Cost: %.6f", userData.TotalCost)

		// Calculate stake amount
		amountStakeFloat := userData.TotalCost * cfg.CostToStakeRate
		amountStakeInt := int64(math.Ceil(amountStakeFloat)) // Round up

		log.Printf(" Conversion rate: %.2f", cfg.CostToStakeRate)
		log.Printf(" Stake amount (before min check): %d %s", amountStakeInt, cfg.StakeUnit)

		// Check minimum amount
		if amountStakeInt < cfg.MinStakeAmount {
			log.Printf(" Stake amount %d is less than minimum %d. Skip.", amountStakeInt, cfg.MinStakeAmount)
			skippedCount++
			log.Println("---------------------------------")
			continue
		}

		// Format amount for command (eg: 15000stake)
		amountStr := fmt.Sprintf("%d%s", amountStakeInt, cfg.StakeUnit)
		log.Printf(" Stake amount sent: %s", amountStr)

		//Caculate payment fee
		feeStakeFloat := float64(amountStakeFloat) * feePercentage
		feeStakeInt := int64(math.Ceil(feeStakeFloat))

		if feeStakeInt < minFeeAmount {
			feeStakeInt = minFeeAmount
		}

		paymentFeeStr := fmt.Sprintf("%d%s", feeStakeInt, cfg.StakeUnit)
		log.Printf("  Payment Fee (1%% of %d, min %d): %s", amountStakeInt, minFeeAmount, paymentFeeStr)

		// Prepare config to send
		streamCfg := streampay.StreamSendConfig{
			StreampaydPath: cfg.StreampaydPath,
			Recipient:      cfg.ProviderAddress,
			Amount:         amountStr,
			Duration:       cfg.StreamDuration,
			ChainID:        cfg.ChainID,
			SenderName:     userID,
			KeyringBackend: cfg.KeyringBackend,
			PaymentFee:     paymentFeeStr,
			DryRun:         cfg.DryRun, // Pass DryRun value
		}

		// Send payment
		log.Printf(" Prepare to send %s to %s (fee: %s)", amountStr, cfg.ProviderAddress, paymentFeeStr)
		txHash, err := streampay.StreamSend(streamCfg)
		if err != nil {
			log.Printf(" [ERROR] Failed to send payment to User %s: %v", userID, err)
			sendErrorCount++
		} else {
			log.Printf(" [SUCCESS] Successfully sent payment to User %s! TxHash: %s", userID, txHash)
			successCount++
		}
		log.Println("---------------------------------")
		// Add a small delay between sends to avoid rate limiting if needed
		// time.Sleep(500 * time.Millisecond)
	}

	// 3. Logging the cycle summary
	log.Printf("===== Payment cycle ended at %s =====", time.Now().Format(time.RFC3339))
	log.Printf("Summary: Success: %d, Skipped (min amount): %d, Address Error: %d, Send Error: %d",
		successCount, skippedCount, addressErrorCount, sendErrorCount)

	// Returns an error if any send or address errors occurred
	if addressErrorCount > 0 || sendErrorCount > 0 {
		return fmt.Errorf("there were %d address errors and %d send errors in cycle", addressErrorCount, sendErrorCount)
	}

	return nil // Returns nil if there is no fatal error
}
