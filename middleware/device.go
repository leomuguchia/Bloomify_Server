package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
