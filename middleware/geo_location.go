// File: middleware/geolocation.go
package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GeoLocation represents the geolocation information for an IP.
type GeoLocation struct {
	IP          string  `json:"ip"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Country     string  `json:"country_name"`
	CountryCode string  `json:"country_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
}

// List of restricted countries.
var restrictedCountries = map[string]bool{
	"North Korea": true,
	"Iran":        true,
}

// geoCache caches geolocation results keyed by IP address.
var geoCache = make(map[string]*GeoLocation)
var cacheMutex sync.RWMutex

// isPrivateIP checks if an IP is private or loopback.
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	privateIPBlocks := []*net.IPNet{
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
	}
	for _, block := range privateIPBlocks {
		if block.Contains(parsedIP) {
			return true
		}
	}
	return false
}

// getGeolocation retrieves geolocation data from an external API (using ipapi.co) and caches the result.
// If the IP is private or the API call fails, it returns a default geolocation with "Unknown" country.
func getGeolocation(ip string, logger *zap.Logger) (*GeoLocation, error) {
	// Check cache first.
	cacheMutex.RLock()
	if geo, exists := geoCache[ip]; exists {
		cacheMutex.RUnlock()
		return geo, nil
	}
	cacheMutex.RUnlock()

	// If the IP is private, return default geolocation.
	if isPrivateIP(ip) {
		logger.Warn("Client IP is private; using default geolocation", zap.String("ip", ip))
		defaultGeo := &GeoLocation{
			IP:      ip,
			Country: "Unknown",
		}
		// Cache default value.
		cacheMutex.Lock()
		geoCache[ip] = defaultGeo
		cacheMutex.Unlock()
		return defaultGeo, nil
	}

	// Query external API.
	url := fmt.Sprintf("https://ipapi.co/%s/json/", ip)
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		logger.Error("Failed to query external geolocation API", zap.String("ip", ip), zap.Error(err))
		return &GeoLocation{
			IP:      ip,
			Country: "Unknown",
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("External geolocation API returned non-OK status", zap.String("ip", ip), zap.Int("status", resp.StatusCode))
		return &GeoLocation{
			IP:      ip,
			Country: "Unknown",
		}, nil
	}

	var geo GeoLocation
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		logger.Error("Failed to decode geolocation response", zap.String("ip", ip), zap.Error(err))
		return &GeoLocation{
			IP:      ip,
			Country: "Unknown",
		}, nil
	}

	// If the external API returned an empty country field, default to "Unknown".
	if geo.Country == "" {
		logger.Warn("Geolocation API returned empty country field; defaulting to Unknown", zap.String("ip", ip))
		geo.Country = "Unknown"
	}

	// Cache the result.
	cacheMutex.Lock()
	geoCache[ip] = &geo
	cacheMutex.Unlock()

	logger.Info("Geolocation retrieved from external API", zap.Any("geo", geo))
	return &geo, nil
}

// GeolocationMiddleware retrieves the client's IP, obtains its geolocation,
// applies restricted region checks, and sets the geolocation in the context.
// If the country is restricted, it aborts the request with a 403 Forbidden.
func GeolocationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := zap.L()

		clientIP := c.ClientIP()
		logger.Info("GeolocationMiddleware: Client IP", zap.String("ip", clientIP))

		if clientIP == "" {
			logger.Error("Client IP is empty; defaulting to Unknown geolocation")
			c.Set("geoLocation", &GeoLocation{
				IP:      clientIP,
				Country: "Unknown",
			})
			c.Next()
			return
		}

		geo, err := getGeolocation(clientIP, logger)
		if err != nil {
			logger.Error("Failed to get geolocation", zap.String("ip", clientIP), zap.Error(err))
			// Use default geolocation if error occurs.
			geo = &GeoLocation{
				IP:      clientIP,
				Country: "Unknown",
			}
		}

		// Block if the country is restricted.
		if restrictedCountries[geo.Country] {
			logger.Warn("GeolocationMiddleware: Blocked request from restricted region", zap.String("country", geo.Country))
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access from your region is restricted"})
			return
		}

		c.Set("geoLocation", geo)
		logger.Info("GeolocationMiddleware: Geolocation determined", zap.Any("geo", geo))
		c.Next()
	}
}
