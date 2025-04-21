package model

import (
	"time"
)

// Window defines the time period for calculating the cost
type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// UserData contains cost information for a user or system
type UserData struct {
	TotalCost      float64
	Window         Window
	NamespaceCosts map[string]float64
}

// CostData represents the entire contents of the JSON file read in
type CostData map[string]interface{}

// ParseUserData processes interface{} data into a specific UserData
// Returns UserData and a boolean indicating whether the parse was successful
func ParseUserData(data interface{}) (UserData, bool) {
	userDataMap, ok := data.(map[string]interface{})
	if !ok {
		return UserData{}, false
	}

	var user UserData
	user.NamespaceCosts = make(map[string]float64)
	foundTotalCost := false
	foundWindow := false

	for key, value := range userDataMap {
		switch key {
		case "totalCost":
			if cost, ok := value.(float64); ok {
				user.TotalCost = cost
				foundTotalCost = true
			}
		case "window":
			// Be more careful when parsing window
			windowInterface, ok := value.(map[string]interface{})
			if !ok {
				continue // Skip if window is not a map
			}
			startStr, okS := windowInterface["start"].(string)
			endStr, okE := windowInterface["end"].(string)

			if okS && okE {
				// Try parsing with RFC3339Nano first, then RFC3339
				start, errS := time.Parse(time.RFC3339Nano, startStr)
				if errS != nil {
					start, errS = time.Parse(time.RFC3339, startStr) // Try formatting without nanoseconds
				}

				end, errE := time.Parse(time.RFC3339Nano, endStr)
				if errE != nil {
					end, errE = time.Parse(time.RFC3339, endStr)
				}

				// Only mark as found if both parses succeed
				if errS == nil && errE == nil {
					user.Window.Start = start
					user.Window.End = end
					foundWindow = true
				} else {
					// Log or handle errors if needed when time parse fails
					// fmt.Printf("Warning: Could not parse window times for key. Start error: %v, End error: %v\n", errS, errE)
				}
			}

		default:
			// Assume remaining keys are namespace cost if float64
			if nsCost, ok := value.(float64); ok {
				user.NamespaceCosts[key] = nsCost
			}
		}
	}

	// Only return true if both totalCost and valid window are found
	return user, foundTotalCost && foundWindow
}
