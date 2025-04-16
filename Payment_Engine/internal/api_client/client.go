package api_client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"payment-engine/internal/model"
)

const defaultTimeout = 30 * time.Second // Timeout for API request

// FetchCostData calls the cost API and parses the response.
// Returns a map with the key being the user ID (or "system") and the value being UserData.
func FetchCostData(apiUrl, window, step string) (map[string]model.UserData, error) {
	// 1. Construct the URL with query parameters
	fullUrl, err := buildUrl(apiUrl, window, step)
	if err != nil {
		return nil, fmt.Errorf("error building API URL: %w", err)
	}
	log.Printf("Fetching cost data from: %s", fullUrl)

	// 2. Make the HTTP GET request
	client := &http.Client{Timeout: defaultTimeout}
	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating API request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Handle network errors (timeout, connection refused, etc.)
		return nil, fmt.Errorf("error executing API request to %s: %w", fullUrl, err)
	}
	defer resp.Body.Close()

	// 3. Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		// Log the response body for debugging
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		log.Printf("API returned non-OK status: %d. Response body: %s", resp.StatusCode, bodyString)
		return nil, fmt.Errorf("API request failed with status code %d", resp.StatusCode)
	}

	// 4. Read and Unmarshal the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading API response body: %w", err)
	}

	if len(bodyBytes) == 0 {
		log.Println("API returned an empty response body.")
		return make(map[string]model.UserData), nil // Return empty map, not an error
	}

	var rawData model.CostData // Reuse the CostData type (map[string]interface{})
	if err := json.Unmarshal(bodyBytes, &rawData); err != nil {
		// Provide context for JSON errors
		log.Printf("Raw JSON response: %s", string(bodyBytes)) // Log raw response on error
		return nil, fmt.Errorf("error parsing JSON response from API: %w", err)
	}

	// 5. Parse raw data into UserData map (using the existing logic from model)
	parsedData := make(map[string]model.UserData)
	for key, value := range rawData {
		userData, ok := model.ParseUserData(value) // Reuse the parser logic
		if ok {
			parsedData[key] = userData
		} else {
			// Log a warning if a specific user's data couldn't be parsed
			log.Printf("Warning: Could not parse valid data for key '%s' from API response\n", key)
		}
	}

	return parsedData, nil
}

// buildUrl constructs the full URL with query parameters safely.
func buildUrl(baseUrl, window, step string) (string, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return "", err
	}

	// Ensure the path is correct (e.g., "/getcost")
	// This assumes the baseUrl might just be "http://localhost:9991"
	// Adjust if baseUrl already contains the path
	if u.Path == "" || u.Path == "/" { // Add path if missing or root
		u.Path = "/getcost" // Or make this configurable if needed
	} else if !strings.HasSuffix(u.Path, "/getcost") {
		// Append if the path exists but doesn't end with /getcost
		// Or handle this case based on expected input for baseUrl
		u.Path = strings.TrimSuffix(u.Path, "/") + "/getcost"
	}

	q := u.Query()
	q.Set("window", window)
	q.Set("step", step)
	u.RawQuery = q.Encode()

	return u.String(), nil
}
