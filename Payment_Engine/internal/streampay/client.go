package streampay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// StreamSendConfig contains parameters specific to the stream-send command
type StreamSendConfig struct {
	StreampaydPath string
	Recipient      string
	Amount         string // Units already exist, e.g. "15000stake"
	Duration       string
	ChainID        string
	SenderName     string
	KeyringBackend string
	PaymentFee     string // 1% of Amount
	DryRun         bool   // Add DryRun here for the client to know
}

// TxResult contains result information from a transaction
type TxResult struct {
	TxHash string `json:"txhash"` // Usually this key in the JSON output
	// Other fields from the JSON output can be added if needed
	RawLog string `json:"raw_log"` // Usually contains error information
	Code   int    `json:"code"`    // Error code (0 is successful)
}

//const commandTimeout = 30 * time.Second // Maximum wait time per CLI command

// // GetRecipientAddress executes 'streampayd keys show' to get the wallet address.
// func GetRecipientAddress(streampaydPath, userID, keyringBackend string) (string, error) {
// 	if userID == "" {
// 		return "", fmt.Errorf("userID cannot be empty")
// 	}
// 	if streampaydPath == "" {
// 		return "", fmt.Errorf("streampayd path cannot be empty")
// 	}

// 	args := []string{
// 		"keys", "show", userID,
// 		"--address", // Get address only
// 		"--keyring-backend", keyringBackend,
// 	}

// 	cmd := exec.Command(streampaydPath, args...)
// 	// Set timeout for the command
// 	// Consider using context with timeout for better control
// 	cmd.WaitDelay = commandTimeout // This attribute doesn't exist directly

// 	var outbuf, errbuf bytes.Buffer
// 	cmd.Stdout = &outbuf
// 	cmd.Stderr = &errbuf

// 	log.Printf("Executing: %s %s\n", streampaydPath, strings.Join(args, " "))

// 	err := cmd.Run() // Run waits for the command to complete

// 	stdout := strings.TrimSpace(outbuf.String())
// 	stderr := strings.TrimSpace(errbuf.String())

// 	if err != nil {
// 		// The error could be due to a key not being found or another execution error
// 		log.Printf("Error running 'keys show' for %s: %v. Stderr: %s\n", userID, err, stderr)
// 		// Return more explicit errors if stderr can be relied upon
// 		if strings.Contains(stderr, "item could not be found") || strings.Contains(stderr, "no such file or directory") {
// 			return "", fmt.Errorf("key '%s' could not be found with backend '%s': %w", userID, keyringBackend, err)
// 		}
// 		return "", fmt.Errorf("error executing 'keys show' for %s (Stderr: %s): %w", userID, stderr, err)
// 	}

// 	// Check stderr even if exit code is 0, because sometimes there is a warning
// 	if stderr != "" {
// 		log.Printf("Warning when running 'keys show' for %s: Stderr: %s\n", userID, stderr)
// 	}

// 	// Check if stdout is empty
// 	if stdout == "" {
// 		return "", fmt.Errorf("'keys show' command for %s ran successfully but did not return address", userID)
// 	}

// 	// Address is usually the only line in stdout when using --address
// 	log.Printf("Address for %s: %s\n", userID, stdout)
// 	return stdout, nil
// }

// StreamSend executes 'streampayd tx streampay stream-send'.
// Returns Tx Hash on success, or error.
func StreamSend(cfg StreamSendConfig) (string, error) {
	if cfg.StreampaydPath == "" {
		return "", fmt.Errorf("streampayd path cannot be empty")
	}
	if cfg.Recipient == "" || cfg.Amount == "" || cfg.Duration == "" || cfg.ChainID == "" || cfg.SenderName == "" || cfg.KeyringBackend == "" || cfg.PaymentFee == "" {
		return "", fmt.Errorf("missing required parameter for StreamSend")
	}
	args := []string{
		"tx", "streampay", "stream-send",
		cfg.Recipient,
		cfg.Amount,
		"--duration", cfg.Duration,
		"--chain-id", cfg.ChainID,
		"--from", cfg.SenderName,
		"--keyring-backend", cfg.KeyringBackend,
		"--payment-fee", cfg.PaymentFee,
		"-y",         // Automatically confirm
		"-o", "json", // Require JSON output for easy parsing
	}

	// Dry Run Mode: Only print the command, do not execute
	if cfg.DryRun {
		log.Printf("[DRY RUN] Command to execute: %s %s\n", cfg.StreampaydPath, strings.Join(args, " "))
		// Return a simulated value that was successful in the dry run
		return fmt.Sprintf("dry-run-tx-for-%s", cfg.Recipient), nil
	}

	cmd := exec.Command(cfg.StreampaydPath, args...)

	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	log.Printf("Executing: %s %s\n", cfg.StreampaydPath, strings.Join(args, " "))

	err := cmd.Run()

	stdout := outbuf.Bytes() // Keep bytes to unmarshal JSON
	stderr := strings.TrimSpace(errbuf.String())

	// Even if there is an execution error (exit code != 0), still try to parse the JSON output
	// because sometimes the error occurs after submitting to the chain and having a txhash
	var result TxResult
	parseErr := json.Unmarshal(stdout, &result)

	if err != nil {
		// Command execution error
		log.Printf("Error running 'stream-send' for %s: %v. Stderr: %s\n", cfg.Recipient, err, stderr)
		// If JSON parse is successful and there is a raw_log, display raw_log first
		if parseErr == nil && result.RawLog != "" {
			log.Printf("Detailed error from raw_log: %s\n", result.RawLog)
			return "", fmt.Errorf("error executing 'stream-send' (code %d): %s", result.Code, result.RawLog)
		}
		// If JSON parse is unsuccessful or there is no raw_log, use stderr
		return "", fmt.Errorf("error executing 'stream-send' (Stderr: %s): %w", stderr, err)
	}

	// Command runs successfully (exit code 0)
	// Check the code in JSON output (0 is success on chain)
	if result.Code != 0 {
		log.Printf("Error in transaction 'stream-send' for %s (code %d). RawLog: %s\n", cfg.Recipient, result.Code, result.RawLog)
		return "", fmt.Errorf("failed transaction (code %d): %s", result.Code, result.RawLog)
	}

	// Success!
	log.Printf("Successfully sent to %s! TxHash: %s\n", cfg.Recipient, result.TxHash)
	if result.TxHash == "" {
		log.Printf("Warning: Transaction successful (code 0) but txhash not found in JSON output.")
		return "unknown-hash-code-0", nil // Returns special value instead of error
	}

	return result.TxHash, nil
}
