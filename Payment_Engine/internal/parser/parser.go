package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	// Replace with your actual module path
	"payment-engine/internal/model"
)

// ParseCostFile reads and parses the cost JSON file.
// Returns a map with the key being the user ID (or "system") and the value being UserData.
func ParseCostFile(filePath string) (map[string]model.UserData, error) {
	dataBytes, err := os.ReadFile(filePath)
	if err != nil {
		// Returns a more specific error if the file does not exist
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("file does not exist: %w", err) // Returns the original error for external checking
		}
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	if len(dataBytes) == 0 {
		// Considers an empty file as a valid case but has no data
		return make(map[string]model.UserData), nil // Returns an empty map
	}

	var rawData model.CostData
	if err := json.Unmarshal(dataBytes, &rawData); err != nil {
		// Provide clearer error information
		var syntaxError *json.SyntaxError
		var typeError *json.UnmarshalTypeError
		if ok := errors.As(err, &syntaxError); ok {
			// Display a small snippet of JSON around the error
			start := max(0, int(syntaxError.Offset)-10)
			end := min(len(dataBytes), int(syntaxError.Offset)+10)
			context := string(dataBytes[start:end])
			return nil, fmt.Errorf("JSON syntax error in file %s at offset %d (near '%s'): %w", filePath, syntaxError.Offset, context, err)
		}
		if ok := errors.As(err, &typeError); ok {
			return nil, fmt.Errorf("error in JSON data type in file %s, field '%s' (offset %d), value '%v', required type '%s': %w", filePath, typeError.Field, typeError.Offset, typeError.Value, typeError.Type, err)
		}
		// Other general unmarshal errors
		return nil, fmt.Errorf("error in parsing JSON file %s: %w", filePath, err)
	}

	parsedData := make(map[string]model.UserData)
	for key, value := range rawData {
		userData, ok := model.ParseUserData(value)
		if ok {
			parsedData[key] = userData
		} else {
			// Warning log if desired
			// fmt.Printf("Warning: Could not parse valid data for key '%s' in file %s\n", key, filePath)
		}
	}

	return parsedData, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
