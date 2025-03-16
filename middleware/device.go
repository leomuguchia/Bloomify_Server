package middleware

import (
	providerRepo "bloomify/database/repository/provider"
	userRepo "bloomify/database/repository/user"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

type IPLocation struct {
	Status     string  `json:"status"`
	Message    string  `json:"message"`
	Country    string  `json:"country"`
	RegionName string  `json:"regionName"`
	City       string  `json:"city"`
	Zip        string  `json:"zip"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	Timezone   string  `json:"timezone"`
	Query      string  `json:"query"`
}

func lookupIPLocation(ip string) (*IPLocation, error) {
	// Use a free geo-IP API endpoint (ip-api.com)
	url := fmt.Sprintf("http://ip-api.com/json/%s", ip)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup IP location: %w", err)
	}
	defer resp.Body.Close()

	var loc IPLocation
	if err := json.NewDecoder(resp.Body).Decode(&loc); err != nil {
		return nil, fmt.Errorf("failed to decode geo response: %w", err)
	}
	if loc.Status != "success" {
		return nil, fmt.Errorf("geo lookup error: %s", loc.Message)
	}
	return &loc, nil
}

func getClientIP(c *gin.Context) string {
	// Check for X-Forwarded-For header (may contain multiple IPs).
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 && ips[0] != "" {
			return strings.TrimSpace(ips[0])
		}
	}
	// Check for X-Real-IP header.
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Fallback: use RemoteAddr (strip port if present).
	ip := c.Request.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

func DeviceDetailsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve device details from custom headers.
		deviceID := c.GetHeader("X-Device-ID")
		deviceName := c.GetHeader("X-Device-Name")
		if deviceID == "" || deviceName == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Missing required device details: X-Device-ID and X-Device-Name",
			})
			return
		}

		// Get the client IP.
		ip := getClientIP(c)

		// Look up location details using the external API.
		loc, err := lookupIPLocation(ip)
		var location string
		if err != nil {
			location = "Unknown"
		} else {
			// Construct a location string (e.g., "City, Region, Country")
			location = fmt.Sprintf("%s, %s, %s", loc.City, loc.RegionName, loc.Country)
		}

		// Set device details in the context for downstream handlers.
		c.Set("deviceID", deviceID)
		c.Set("deviceName", deviceName)
		c.Set("deviceIP", ip)
		c.Set("deviceLocation", location)

		c.Next()
	}
}

func DeviceAuthMiddlewareUser(userRepo userRepo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the device ID from context (set by DeviceDetailsMiddleware)
		deviceID, exists := c.Get("deviceID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing device details"})
			return
		}

		// Retrieve the user ID from context (set by JWTAuthUserMiddleware)
		userID, exists := c.Get("userID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}

		// Use a projection to fetch only the devices field.
		projection := bson.M{"devices": 1}
		user, err := userRepo.GetByIDWithProjection(userID.(string), projection)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}

		// Check if the device exists in the user's devices list.
		deviceFound := false
		for _, d := range user.Devices {
			if d.DeviceID == deviceID.(string) {
				deviceFound = true
				break
			}
		}

		if !deviceFound {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Device not recognized"})
			return
		}

		c.Next()
	}
}

func DeviceAuthMiddlewareProvider(providerRepo providerRepo.ProviderRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID, exists := c.Get("deviceID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing device details"})
			return
		}

		providerID, exists := c.Get("providerID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Provider not authenticated"})
			return
		}

		// Use a projection to fetch only the devices field.
		projection := bson.M{"devices": 1}
		provider, err := providerRepo.GetByIDWithProjection(providerID.(string), projection)
		if err != nil || provider == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Provider not found"})
			return
		}

		// Check if the device exists in the provider's devices list.
		deviceFound := false
		for _, d := range provider.Devices {
			if d.DeviceID == deviceID.(string) {
				deviceFound = true
				break
			}
		}

		if !deviceFound {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Device not recognized"})
			return
		}

		c.Next()
	}
}
