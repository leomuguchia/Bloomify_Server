package utils

import (
	"bloomify/config"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

type ExchangeRateAPIResponse struct {
	Result string             `json:"result"`
	Base   string             `json:"base_code"`
	Rates  map[string]float64 `json:"conversion_rates"`
}

// fetchExchangeRate fetches exchange rate from base to target using ExchangeRate-API.
func fetchExchangeRate(from, to string) (float64, error) {
	url := fmt.Sprintf("https://v6.exchangerate-api.com/v6/%s/latest/%s", config.AppConfig.ExchangeRateAPIKey, from)

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
		return 0, fmt.Errorf("exchange API returned failure result")
	}

	rate, ok := rateResp.Rates[to]
	if !ok {
		return 0, fmt.Errorf("exchange rate for %s not found", to)
	}
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

func GetGeoPricingBias(countryCode string) float64 {
	if bias, ok := config.CountryBiasMap[strings.ToUpper(countryCode)]; ok {
		return bias
	}
	return 1.0
}
