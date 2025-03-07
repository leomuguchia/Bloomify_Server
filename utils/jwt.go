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

// GenerateToken creates a signed JWT token with the given subject (e.g., providerID or userID) and email.
// The token expires after the specified duration.
func GenerateToken(subject, email string, duration time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":   subject,
		"email": email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(duration).Unix(),
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

// ExtractProviderIDFromToken extracts the ID (subject) from a valid JWT token string.
// It returns the extracted ID or an error if validation fails.
func ExtractIDFromToken(tokenString string) (string, error) {
	token, err := ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", errors.New("token does not contain a valid 'sub' claim")
	}

	return sub, nil
}
