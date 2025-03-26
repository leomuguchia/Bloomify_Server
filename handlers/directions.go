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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var directions DirectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&directions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
