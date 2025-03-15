package user

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	googlePublicKeys  map[string]*rsa.PublicKey
	googleKeysMutex   sync.RWMutex
	googleKeysExpires time.Time
)

// getGooglePublicKeys fetches and caches Google’s public keys.
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Google certs: %w", err)
	}

	var certs struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &certs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Google certs: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, key := range certs.Keys {
		pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(createPEM(key.N, key.E)))
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %w", err)
		}
		keys[key.Kid] = pubKey
	}

	googleKeysMutex.Lock()
	googlePublicKeys = keys
	// Set an expiration (for example, 1 hour)
	googleKeysExpires = time.Now().Add(1 * time.Hour)
	googleKeysMutex.Unlock()

	return keys, nil
}

// createPEM is a helper to construct a PEM from modulus and exponent.
// In production, you would use a proper conversion library.
func createPEM(n, e string) string {
	// This function should convert the base64url-encoded n and e into an RSA public key PEM format.
	// For brevity, this is left as a placeholder.
	return "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
}

// ValidateSocialToken validates a token for a given provider.
func ValidateSocialToken(provider, tokenStr string) error {
	switch provider {
	case "google":
		return validateGoogleToken(tokenStr)
	case "apple":
		return validateAppleToken(tokenStr)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

func validateGoogleToken(tokenStr string) error {
	keys, err := getGooglePublicKeys()
	if err != nil {
		return fmt.Errorf("failed to get Google public keys: %w", err)
	}
	// Parse the token without verifying first to extract the kid.
	token, err := jwt.Parse(tokenStr, nil)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}
	if token.Header["kid"] == nil {
		return errors.New("token missing kid header")
	}
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return errors.New("invalid kid type")
	}
	pubKey, exists := keys[kid]
	if !exists {
		return errors.New("no matching public key found")
	}
	// Now parse and verify the token with the proper key.
	parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method.
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		return fmt.Errorf("failed to verify token: %w", err)
	}
	if !parsedToken.Valid {
		return errors.New("invalid token")
	}
	return nil
}

func validateAppleToken(tokenStr string) error {
	// Apple token validation requires downloading Apple's public keys and similar logic.
	// For production, use a well-tested library or refer to Apple's documentation.
	// For brevity, we simulate validation here.
	// You would typically:
	// 1. Fetch Apple's public keys from https://appleid.apple.com/auth/keys
	// 2. Parse the token and verify using the correct public key.
	// 3. Return an error if the token is invalid.
	return nil
}
