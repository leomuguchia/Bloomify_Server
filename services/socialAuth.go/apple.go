package socialAuth

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	applePublicKeys  map[string]*rsa.PublicKey
	appleKeysMutex   sync.RWMutex
	appleKeysExpires time.Time
)

// AppleJWK represents a single JSON Web Key from Apple's keys endpoint.
type AppleJWK struct {
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// AppleJWKResponse represents the response from Apple's keys endpoint.
type AppleJWKResponse struct {
	Keys []AppleJWK `json:"keys"`
}

// getApplePublicKeys fetches and caches Apple's public keys.
func getApplePublicKeys() (map[string]*rsa.PublicKey, error) {
	appleKeysMutex.RLock()
	if time.Now().Before(appleKeysExpires) && applePublicKeys != nil {
		defer appleKeysMutex.RUnlock()
		return applePublicKeys, nil
	}
	appleKeysMutex.RUnlock()

	resp, err := http.Get("https://appleid.apple.com/auth/keys")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Apple public keys: %w", err)
	}
	defer resp.Body.Close()

	var keyResp AppleJWKResponse
	if err := json.NewDecoder(resp.Body).Decode(&keyResp); err != nil {
		return nil, fmt.Errorf("failed to decode Apple keys: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, key := range keyResp.Keys {
		pubKey, err := convertJWKToPublicKey(key.N, key.E)
		if err != nil {
			return nil, fmt.Errorf("failed to convert JWK to public key: %w", err)
		}
		keys[key.Kid] = pubKey
	}

	appleKeysMutex.Lock()
	applePublicKeys = keys
	// Cache keys for 24 hours (Apple keys don’t rotate that often)
	appleKeysExpires = time.Now().Add(24 * time.Hour)
	appleKeysMutex.Unlock()

	return keys, nil
}

// ValidateAppleToken validates the Apple ID token and returns user info (email and name if available).
func ValidateAppleToken(tokenStr string, audience string) (*UserInfo, error) {
	keys, err := getApplePublicKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to get Apple public keys: %w", err)
	}

	parser := new(jwt.Parser)
	unverifiedToken, _, err := parser.ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	kid, ok := unverifiedToken.Header["kid"].(string)
	if !ok {
		return nil, errors.New("token missing kid header")
	}

	pubKey, exists := keys[kid]
	if !exists {
		return nil, errors.New("no matching Apple public key found")
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid Apple ID token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("failed to parse claims")
	}

	// Verify audience claim matches your client ID (app bundle ID)
	if aud, ok := claims["aud"].(string); !ok || aud != audience {
		return nil, errors.New("invalid audience in Apple ID token")
	}

	// Verify issuer claim
	if iss, ok := claims["iss"].(string); !ok || iss != "https://appleid.apple.com" {
		return nil, errors.New("invalid issuer in Apple ID token")
	}

	// Verify expiry
	if exp, ok := claims["exp"].(float64); !ok || int64(exp) < time.Now().Unix() {
		return nil, errors.New("apple ID token expired")
	}

	email, emailOk := claims["email"].(string)
	if !emailOk {
		return nil, errors.New("email claim not found in Apple ID token")
	}

	email = strings.ToLower(email)
	name := ""
	if nameClaim, ok := claims["name"].(string); ok {
		name = nameClaim
	}

	return &UserInfo{
		Email: email,
		Name:  name,
	}, nil
}
