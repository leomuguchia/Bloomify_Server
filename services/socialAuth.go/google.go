package socialAuth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	googlePublicKeys  map[string]*rsa.PublicKey
	googleKeysMutex   sync.RWMutex
	googleKeysExpires time.Time
)

// GoogleJWK represents a single JSON Web Key from Google's keys endpoint.
type GoogleJWK struct {
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// GoogleJWKResponse represents the response from Google's keys endpoint.
type GoogleJWKResponse struct {
	Keys []GoogleJWK `json:"keys"`
}

// UserInfo holds extracted user info from tokens.
type UserInfo struct {
	Email string
	Name  string
}

// getGooglePublicKeys fetches and caches Google's public keys.
func getGooglePublicKeys() (map[string]*rsa.PublicKey, error) {
	googleKeysMutex.RLock()
	if time.Now().Before(googleKeysExpires) && googlePublicKeys != nil {
		defer googleKeysMutex.RUnlock()
		return googlePublicKeys, nil
	}
	googleKeysMutex.RUnlock()

	resp, err := http.Get("https://www.googleapis.com/oauth2/v3/certs")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google certs: %w", err)
	}
	defer resp.Body.Close()

	var keyResp GoogleJWKResponse
	if err := json.NewDecoder(resp.Body).Decode(&keyResp); err != nil {
		return nil, fmt.Errorf("failed to decode Google keys: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, key := range keyResp.Keys {
		pubKey, err := convertJWKToPublicKey(key.N, key.E)
		if err != nil {
			return nil, fmt.Errorf("failed to convert JWK to public key: %w", err)
		}
		keys[key.Kid] = pubKey
	}

	googleKeysMutex.Lock()
	googlePublicKeys = keys
	// Cache keys for 1 hour (Google rotates keys frequently)
	googleKeysExpires = time.Now().Add(1 * time.Hour)
	googleKeysMutex.Unlock()

	return keys, nil
}

// convertJWKToPublicKey converts base64url encoded modulus and exponent to rsa.PublicKey.
func convertJWKToPublicKey(n, e string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}
	eb, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert exponent bytes to int
	var exp int
	for _, b := range eb {
		exp = exp<<8 + int(b)
	}

	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nb),
		E: exp,
	}
	return pubKey, nil
}

// ValidateGoogleToken validates the Google ID token and returns user info (email and name).
func ValidateGoogleToken(tokenStr string, audience string) (*UserInfo, error) {
	keys, err := getGooglePublicKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to get Google public keys: %w", err)
	}

	// Parse token without verification to get the kid from header
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
		return nil, errors.New("no matching Google public key found")
	}

	// Parse and verify token using the right public key
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
		return nil, errors.New("invalid Google ID token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("failed to parse claims")
	}

	// Verify audience claim matches your client ID (Google OAuth client ID)
	if aud, ok := claims["aud"].(string); !ok || aud != audience {
		return nil, errors.New("invalid audience in Google ID token")
	}

	// Verify issuer claim
	if iss, ok := claims["iss"].(string); !ok || (iss != "accounts.google.com" && iss != "https://accounts.google.com") {
		return nil, errors.New("invalid issuer in Google ID token")
	}

	// Verify expiry
	if exp, ok := claims["exp"].(float64); !ok || int64(exp) < time.Now().Unix() {
		return nil, errors.New("google ID token expired")
	}

	email, emailOk := claims["email"].(string)
	if !emailOk {
		return nil, errors.New("email claim not found in Google ID token")
	}

	email = strings.ToLower(email)
	name, _ := claims["name"].(string)

	return &UserInfo{
		Email: email,
		Name:  name,
	}, nil
}
