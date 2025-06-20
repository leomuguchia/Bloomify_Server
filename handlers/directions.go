package handlers

import (
	"bloomify/config"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DirectionsResponse represents the structure of the response from Google Directions API.
type DirectionsResponse struct {
	Routes []struct {
		OverviewPolyline struct {
			Points string `json:"points"`
		} `json:"overview_polyline"`
	} `json:"routes"`
}

// GetDirections fetches directions from Google and returns the polyline.
func (hb *BookingHandler) GetDirections(c *gin.Context) {
	originLat := c.Query("originLat")
	originLng := c.Query("originLng")
	destLat := c.Query("destLat")
	destLng := c.Query("destLng")

	// Validate query parameters.
	if originLat == "" || originLng == "" || destLat == "" || destLng == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required query parameters: originLat, originLng, destLat, destLng"})
		return
	}

	// Retrieve API key from configuration.
	apiKey := config.AppConfig.GoogleAPIKey
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "API authentication error"})
		return
	}

	// Build URL for Google Directions API.
	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/directions/json?origin=%s,%s&destination=%s,%s&key=%s",
		originLat, originLng, destLat, destLng, apiKey,
	)

	// Make the API request.
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Please try again later"})
		return
	}
	defer resp.Body.Close()

	var directions DirectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&directions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Please try again later"})
		return
	}

	if len(directions.Routes) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No route found"})
		return
	}

	// Return the polyline for the first route.
	polyline := directions.Routes[0].OverviewPolyline.Points
	c.JSON(http.StatusOK, gin.H{"polyline": polyline})
}

func (hb *BookingHandler) GeocodeAddress(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required query parameter: address"})
		return
	}

	apiKey := config.AppConfig.GoogleAPIKey
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "API authentication error"})
		return
	}

	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s",
		address, apiKey,
	)

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Geocoding request failed"})
		return
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode geocoding response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": data["results"]})
}

func (hb *BookingHandler) ReverseGeocode(c *gin.Context) {
	latitude := c.Query("latitude")
	longitude := c.Query("longitude")

	if latitude == "" || longitude == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required query parameters: latitude, longitude"})
		return
	}

	apiKey := config.AppConfig.GoogleAPIKey
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "API authentication error"})
		return
	}

	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/geocode/json?latlng=%s,%s&key=%s",
		latitude, longitude, apiKey,
	)

	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reverse geocoding request failed"})
		return
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode reverse geocoding response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": data["results"]})
}
