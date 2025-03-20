package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

// Load the secret from an environment variable. Fallback to a default (not recommended in production).
var secretKey = []byte(getSecret())

func getSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "MUGUCHIA"
	}
	return secret
}

// GenerateToken creates a signed JWT token with the given subject (userID),
// email, and deviceID. The token includes both IDs in its claims and does not expire.
func GenerateToken(subject, email, deviceID string) (string, error) {
	claims := jwt.MapClaims{
		"sub":       subject,
		"email":     email,
		"device_id": deviceID,
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// HashToken computes a SHA-256 hash of the token string.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ValidateToken parses and validates a token string and returns the token if valid.
func ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure that the token's signing method is HMAC.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secretKey, nil
	})
}

// ExtractIDsFromToken extracts both the user ID (subject) and device ID from a valid JWT token string.
// It returns the extracted IDs or an error if validation fails.
func ExtractIDsFromToken(tokenString string) (userID, deviceID string, err error) {
	token, err := ValidateToken(tokenString)
	if err != nil {
		return "", "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", errors.New("invalid token")
	}

	uid, ok := claims["sub"].(string)
	if !ok || uid == "" {
		return "", "", errors.New("token does not contain a valid 'sub' claim")
	}

	did, ok := claims["device_id"].(string)
	if !ok || did == "" {
		return "", "", errors.New("token does not contain a valid 'device_id' claim")
	}

	return uid, did, nil
}
