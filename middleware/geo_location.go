// File: middleware/geolocation.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// A sample list of restricted countries.
var restrictedCountries = map[string]bool{
	"North Korea": true,
	"Iran":        true,
}

// A sample list of Tor exit node IPs (for demonstration purposes).
var torExitNodes = map[string]bool{
	"1.2.3.4": true, // replace with real Tor exit IPs or integrate a live list
}

// mockGetCountryFromIP simulates determining the country from an IP address.
// In production, replace this with a real IP geolocation service.
func mockGetCountryFromIP(ip string) (string, error) {
	// For demonstration: if IP starts with "203.", we say it's from "North Korea".
	if strings.HasPrefix(ip, "203.") {
		return "North Korea", nil
	}
	// Otherwise assume it's "USA"
	return "USA", nil
}

// IsTorExitNode checks if the given IP is known as a Tor exit node.
func IsTorExitNode(ip string) bool {
	// In production, use a reliable service or an updated list.
	_, exists := torExitNodes[ip]
	return exists
}

// GeolocationMiddleware blocks requests from restricted regions or Tor exit nodes.
func GeolocationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := zap.L()
		clientIP := getClientIP(c)
		logger.Info("Client IP", zap.String("ip", clientIP))

		// Check if the IP is a known Tor exit node.
		if IsTorExitNode(clientIP) {
			logger.Warn("Blocked request from Tor exit node", zap.String("ip", clientIP))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access from Tor networks is not allowed"})
			return
		}

		// Determine the country from IP (simulate lookup).
		country, err := mockGetCountryFromIP(clientIP)
		if err != nil {
			logger.Error("Failed to get country from IP", zap.String("ip", clientIP), zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unable to determine geolocation"})
			return
		}
		logger.Info("Country determined", zap.String("country", country))

		// Block if the country is restricted.
		if restrictedCountries[country] {
			logger.Warn("Blocked request from restricted region", zap.String("country", country))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access from your region is restricted"})
			return
		}

		c.Next()
	}
}
