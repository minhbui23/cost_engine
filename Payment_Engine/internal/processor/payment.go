package processor

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	std "github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"payment-engine/internal/api_client"
	"payment-engine/internal/config"
	"payment-engine/internal/streampay"
)

const (
	feePercentage = 0.01 // Vẫn giữ để log, nhưng không đưa vào tx
	minFeeAmount  = 1    // Vẫn giữ để log, nhưng không đưa vào tx
)

var ModuleBasics = module.NewBasicManager()

// setupInterfaceRegistry configures the InterfaceRegistry with necessary types.
func setupInterfaceRegistry() types.InterfaceRegistry {
	interfaceRegistry := codectypes.NewInterfaceRegistry()

	std.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	authtypes.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	return interfaceRegistry
}

func setupCodec(registry types.InterfaceRegistry) codec.ProtoCodecMarshaler {
	return codec.NewProtoCodec(registry)
}

// RunPaymentCycle performs a complete payment processing cycle using gRPC for bank transfers.
func RunPaymentCycle(cfg config.Config) error {
	log.Printf("===== Start payment cycle at %s =====", time.Now().Format(time.RFC3339))
	if cfg.DryRun {
		log.Println("[DRY RUN MODE ENABLED] Will not execute gRPC calls.")
	}

	// --- Initialize gRPC Client and Codec ---
	registry := setupInterfaceRegistry()
	cdc := setupCodec(registry)
	grpcClient, err := streampay.NewGrpcClient(cfg, registry, cdc)
	if err != nil {
		log.Printf("[FATAL ERROR] Failed to initialize gRPC client: %v", err)
		log.Printf("===== End of cycle (gRPC init error) at %s =====", time.Now().Format(time.RFC3339))
		return fmt.Errorf("gRPC client initialization failed: %w", err)
	}
	defer grpcClient.Close()

	// 1. Fetch cost data from API
	log.Printf("Fetching cost data from API: %s (Window: %s, Step: %s)", cfg.ApiUrl, cfg.ApiWindow, cfg.ApiStep)
	costData, err := api_client.FetchCostData(cfg.ApiUrl, cfg.ApiWindow, cfg.ApiStep)
	if err != nil {
		log.Printf("[FATAL ERROR] Failed to fetch or parse cost data from API: %v", err)
		log.Printf("===== End of cycle (API error) at %s =====", time.Now().Format(time.RFC3339))
		return fmt.Errorf("error fetching cost data from API: %w", err)
	}
	if len(costData) == 0 {
		log.Println("API returned no user cost data. End of cycle.")
		log.Printf("===== End of cycle (no data) at %s =====", time.Now().Format(time.RFC3339))
		return nil
	}
	log.Printf("Successfully fetched and parsed %d items from API.", len(costData))

	// 2. Loop through each user and process
	successCount := 0
	skippedCount := 0
	keyErrorCount := 0
	sendErrorCount := 0

	// --- Get Provider Address ---
	providerSdkAddr, err := sdk.AccAddressFromBech32(cfg.ProviderAddress)
	if err != nil {
		log.Printf("[FATAL ERROR] Invalid provider address '%s': %v", cfg.ProviderAddress, err)
		return fmt.Errorf("invalid provider address: %w", err)
	}
	log.Printf("Provider address: %s", providerSdkAddr.String())

	for userID, userData := range costData {
		if strings.ToLower(userID) == "system" {
			continue
		}

		log.Printf("--- Processing User: %s ---", userID)
		log.Printf(" Original Cost: %.6f", userData.TotalCost)

		// Get private key from MNEMONIC
		var senderPrivKey cryptotypes.PrivKey
		mnemonicFilename := fmt.Sprintf("%s_MNEMONIC.txt", userID)
		mnemonicPath := filepath.Join(cfg.KeyDirectory, mnemonicFilename)

		// Get content of mnemonic file
		log.Printf(" Reading mnemonic file: %s", mnemonicPath)
		mnemonicBytes, err := os.ReadFile(mnemonicPath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf(" [ERROR] Mnemonic file not found for User %s: %s", userID, mnemonicPath)
			} else {
				log.Printf(" [ERROR] Failed to read mnemonic file for User %s (%s): %v", userID, mnemonicPath, err)
			}
			keyErrorCount++
			log.Println("---------------------------------")
			continue
		}

		// Get mnemonic string from bytes
		mnemonic := strings.TrimSpace(string(mnemonicBytes))
		if mnemonic == "" {
			log.Printf(" [ERROR] Mnemonic file is empty for User %s: %s", userID, mnemonicPath)
			keyErrorCount++
			log.Println("---------------------------------")
			continue
		}

		// 4. Derive Private Key từ Mnemonic (Đã fix cho Cosmos SDK v0.47.9)
		log.Printf(" Deriving private key from mnemonic for User %s...", userID)

		fullPath := sdk.GetConfig().GetFullBIP44Path()

		derivedRawPrivKey, err := hd.Secp256k1.Derive()(mnemonic, "", fullPath)
		if err != nil {
			log.Printf(" [ERROR] Failed to derive raw key for User %s using path %s: %v", userID, fullPath, err)
			keyErrorCount++
			log.Println("---------------------------------")
			continue
		}

		senderPrivKey = hd.Secp256k1.Generate()(derivedRawPrivKey)

		senderAddr := sdk.AccAddress(senderPrivKey.PubKey().Address())
		log.Printf(" Derived key for User %s. Address: %s", userID, senderAddr.String())

		// Calculate transfer amount
		amountStakeFloat := userData.TotalCost * cfg.CostToStakeRate
		amountStakeInt := int64(math.Ceil(amountStakeFloat))

		log.Printf(" Conversion rate: %.2f", cfg.CostToStakeRate)
		log.Printf(" Transfer amount (before min check): %d %s", amountStakeInt, cfg.StakeUnit)

		if amountStakeInt < cfg.MinStakeAmount {
			log.Printf(" Transfer amount %d is less than minimum %d. Skip.", amountStakeInt, cfg.MinStakeAmount)
			skippedCount++
			log.Println("---------------------------------")
			continue
		}

		// Create sdk.Coin for transfer amount
		amountCoin := sdk.NewCoin(cfg.StakeUnit, sdk.NewInt(amountStakeInt))
		log.Printf(" Transfer amount to send: %s", amountCoin.String())

		// Calculate payment fee (chỉ để log, không dùng trong tx)
		feeStakeFloat := float64(amountStakeFloat) * feePercentage
		feeStakeInt := int64(math.Ceil(feeStakeFloat))
		if feeStakeInt < minFeeAmount {
			feeStakeInt = minFeeAmount
		}
		_ = sdk.NewCoin(cfg.StakeUnit, sdk.NewInt(feeStakeInt)) // Tính nhưng không gán vào đâu cả
		// log.Printf("  (Informational) Calculated Payment Fee (1%% of %d, min %d): %s", amountStakeInt, minFeeAmount, paymentFeeCoin.String()) // Log nếu muốn

		// Create sdk.Coin for gas fee
		gasFeeCoin := sdk.NewCoin(cfg.GasFeeDenom, sdk.NewInt(cfg.GasFeeAmount))
		log.Printf("  Gas Fee: %s", gasFeeCoin.String())
		log.Printf("  Gas Limit: %d", cfg.GasLimit)

		// Prepare parameters for gRPC bank transfer call
		sendParams := streampay.SendTxParams{
			SenderPrivateKey: senderPrivKey,
			RecipientAddress: providerSdkAddr.String(),
			Amount:           amountCoin,
			GasLimit:         cfg.GasLimit,
			GasFee:           gasFeeCoin,
			Memo:             fmt.Sprintf("Payment for user %s", userID),
		}

		// Send payment via gRPC (bank transfer)
		log.Printf(" Prepare to send %s to %s via gRPC (Bank Transfer - Tx Fee: %s, Gas: %d)",
			amountCoin.String(), sendParams.RecipientAddress, gasFeeCoin.String(), cfg.GasLimit)

		// --- Thực hiện gửi nếu không phải DryRun ---
		var txResponse *sdk.TxResponse
		var sendErr error

		if cfg.DryRun {
			log.Println(" [DRY RUN] Skipping gRPC BroadcastTx call.")
			txResponse = &sdk.TxResponse{
				TxHash: fmt.Sprintf("dry-run-tx-hash-for-%s", userID),
				Code:   0,
			}
			sendErr = nil
		} else {
			txResponse, sendErr = grpcClient.SendBankTransferViaGrpc(sendParams)
		}
		// -----------------------------------------

		if sendErr != nil {
			log.Printf(" [ERROR] Failed to send payment to User %s via gRPC: %v", userID, sendErr)
			if txResponse != nil {
				log.Printf("   TxResponse Code: %d, RawLog: %s", txResponse.Code, txResponse.RawLog)
			}
			sendErrorCount++
		} else if txResponse != nil {
			log.Printf(" [SUCCESS] Successfully sent payment for User %s! TxHash: %s", userID, txResponse.TxHash)
			successCount++
		} else if !cfg.DryRun {
			log.Printf(" [ERROR] SendBankTransferViaGrpc returned nil response and nil error for User %s.", userID)
			sendErrorCount++
		}
		// --------------------

		log.Println("---------------------------------")
	}

	// 3. Logging the cycle summary
	log.Printf("===== Payment cycle ended at %s =====", time.Now().Format(time.RFC3339))
	log.Printf("Summary: Success: %d, Skipped (min amount): %d, Key Error: %d, Send Error: %d",
		successCount, skippedCount, keyErrorCount, sendErrorCount)

	// Returns an error if any key or send errors occurred
	if keyErrorCount > 0 || sendErrorCount > 0 {
		return fmt.Errorf("there were %d key errors and %d send errors in cycle", keyErrorCount, sendErrorCount)
	}

	return nil
}
