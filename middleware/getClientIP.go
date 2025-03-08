package middleware

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header, which can contain multiple IPs.
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// The header may contain a comma-separated list of IPs. Use the first one.
		ips := strings.Split(xff, ",")
		if len(ips) > 0 && ips[0] != "" {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header.
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback: use the remote address.
	ip := c.Request.RemoteAddr
	// RemoteAddr might be in "ip:port" format; strip the port if present.
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}
