package middleware

import (
	providerRepo "bloomify/database/repository/provider"
	userRepo "bloomify/database/repository/user"
	"bloomify/models"
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
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 && ips[0] != "" {
			return strings.TrimSpace(ips[0])
		}
	}
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	ip := c.Request.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}

func DeviceDetailsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.GetHeader("X-Device-ID")
		deviceName := c.GetHeader("X-Device-Name")
		if deviceID == "" || deviceName == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Missing required device details: X-Device-ID and X-Device-Name",
			})
			return
		}
		ip := getClientIP(c)
		loc, err := lookupIPLocation(ip)
		location := "Unknown"
		if err == nil {
			location = fmt.Sprintf("%s, %s, %s", loc.City, loc.RegionName, loc.Country)
		}
		c.Set("deviceID", deviceID)
		c.Set("deviceName", deviceName)
		c.Set("deviceIP", ip)
		c.Set("deviceLocation", location)
		c.Next()
	}
}

func DeviceAuthMiddlewareUser(userRepo userRepo.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID, exists := c.Get("deviceID")
		if !exists || deviceID == nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing device details", "code": 0})
			return
		}
		userID, exists := c.Get("userID")
		if !exists || userID == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "code": 0})
			return
		}
		projection := bson.M{"devices": 1}
		user, err := userRepo.GetByIDWithProjection(userID.(string), projection)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User not found", "code": 0})
			return
		}
		if user.Devices == nil {
			user.Devices = []models.Device{}
		}
		deviceFound := false
		for _, d := range user.Devices {
			if d.DeviceID == deviceID.(string) {
				deviceFound = true
				break
			}
		}
		if !deviceFound {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Device not recognized", "code": 0})
			return
		}
		c.Next()
	}
}

func DeviceAuthMiddlewareProvider(providerRepo providerRepo.ProviderRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID, exists := c.Get("deviceID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Missing device details", "code": 0})
			return
		}
		providerID, exists := c.Get("providerID")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Provider not authenticated", "code": 0})
			return
		}
		projection := bson.M{"devices": 1}
		provider, err := providerRepo.GetByIDWithProjection(providerID.(string), projection)
		if err != nil || provider == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Provider not found", "code": 0})
			return
		}
		deviceFound := false
		for _, d := range provider.Devices {
			if d.DeviceID == deviceID.(string) {
				deviceFound = true
				break
			}
		}
		if !deviceFound {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Device not recognized", "code": 0})
			return
		}
		c.Next()
	}
}
