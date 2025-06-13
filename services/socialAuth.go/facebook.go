package socialAuth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// FacebookUserInfo holds extracted user info from Facebook
type FacebookUserInfo struct {
	Email string
	Name  string
	ID    string
}

// ValidateFacebookToken validates the Facebook access token and returns user info
func ValidateFacebookToken(accessToken string, appID string, appSecret string) (*FacebookUserInfo, error) {
	// First verify the token is valid and get the user ID
	appAccessToken := fmt.Sprintf("%s|%s", appID, appSecret)

	// Verify the token
	verifyURL := fmt.Sprintf("https://graph.facebook.com/debug_token?input_token=%s&access_token=%s",
		accessToken, appAccessToken)

	resp, err := http.Get(verifyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to verify Facebook token: %w", err)
	}
	defer resp.Body.Close()

	var verifyResult struct {
		Data struct {
			AppID       string `json:"app_id"`
			IsValid     bool   `json:"is_valid"`
			UserID      string `json:"user_id"`
			ExpiresAt   int64  `json:"expires_at"`
			Application string `json:"application"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&verifyResult); err != nil {
		return nil, fmt.Errorf("failed to decode Facebook verify response: %w", err)
	}

	if !verifyResult.Data.IsValid {
		return nil, errors.New("invalid Facebook token")
	}

	if verifyResult.Data.AppID != appID {
		return nil, errors.New("token was issued for a different app")
	}

	// Check if token is expired
	if time.Now().Unix() > verifyResult.Data.ExpiresAt {
		return nil, errors.New("Facebook token has expired")
	}

	// Now get user info
	userInfoURL := fmt.Sprintf("https://graph.facebook.com/v12.0/me?fields=id,name,email&access_token=%s",
		accessToken)

	userResp, err := http.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get Facebook user info: %w", err)
	}
	defer userResp.Body.Close()

	var userInfo struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode Facebook user info: %w", err)
	}

	if userInfo.Email == "" {
		return nil, errors.New("email permission not granted or not available")
	}

	return &FacebookUserInfo{
		Email: strings.ToLower(userInfo.Email),
		Name:  userInfo.Name,
		ID:    userInfo.ID,
	}, nil
}
