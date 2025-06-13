package utils

import (
	"bloomify/config"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ExchangeRateAPIResponse struct {
	Result string             `json:"result"`
	Base   string             `json:"base_code"`
	Rates  map[string]float64 `json:"conversion_rates"`
}

var exchangeRateCache sync.Map // key: "FROM->TO", value: float64

const exchangeRateAPIKey = "7b320b62a306046a3c202b48"

func fetchExchangeRate(from, to string) (float64, error) {
	cacheKey := fmt.Sprintf("%s->%s", from, to)

	if val, ok := exchangeRateCache.Load(cacheKey); ok {
		return val.(float64), nil
	}

	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/%s", exchangeRateAPIKey, from)
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	var rateResp ExchangeRateAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&rateResp); err != nil {
		return 0, fmt.Errorf("decoding response failed: %w", err)
	}

	if rateResp.Result != "success" {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("exchange API returned failure result: status %d, body: %s", resp.StatusCode, string(body))
	}

	rate, ok := rateResp.Rates[to]
	if !ok {
		return 0, fmt.Errorf("exchange rate for %s not found", to)
	}

	exchangeRateCache.Store(cacheKey, rate)
	return rate, nil
}

// ConvertCurrency converts amount between currencies using live rates.
func ConvertCurrency(amount float64, fromCurrency, toCurrency string) (float64, error) {
	if fromCurrency == toCurrency {
		return math.Round(amount*100) / 100, nil
	}
	rate, err := fetchExchangeRate(fromCurrency, toCurrency)
	if err != nil {
		return 0, err
	}
	converted := amount * rate
	return math.Round(converted*100) / 100, nil
}

var TierMultipliers = map[string]float64{
	"tier1": 1.0,
	"tier2": 0.6,
	"tier3": 0.3,
}

var (
	ErrEmptyCountryCode = errors.New("country code cannot be empty")
	ErrCountryNotFound  = errors.New("country code not found in pricing tiers")
)

func GetGeoPricingBias(countryCode string, optionalBiasMap ...map[string]map[string]float64) (float64, error) {
	// Strict validation - empty country code is an error
	if strings.TrimSpace(countryCode) == "" {
		return 0, ErrEmptyCountryCode
	}

	normalizedCode := strings.ToUpper(strings.TrimSpace(countryCode))

	// Determine which bias map to use
	var biasMap map[string]map[string]float64
	if len(optionalBiasMap) > 0 && optionalBiasMap[0] != nil {
		biasMap = optionalBiasMap[0]
	} else {
		biasMap = config.CountryBiasMap // Fallback to server config
	}

	// First try exact match with normalized code
	for tier, countries := range biasMap {
		if bias, ok := countries[normalizedCode]; ok {
			if tierMultiplier, exists := TierMultipliers[tier]; exists {
				return bias * tierMultiplier, nil
			}
			return bias, nil
		}
	}

	// Fallback to case-insensitive search
	for tier, countries := range biasMap {
		for code, bias := range countries {
			if strings.EqualFold(code, normalizedCode) {
				if tierMultiplier, exists := TierMultipliers[tier]; exists {
					return bias * tierMultiplier, nil
				}
				return bias, nil
			}
		}
	}

	// No matches found - return error instead of default value
	return 0, ErrCountryNotFound
}
