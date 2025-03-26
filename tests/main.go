package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"bloomify/utils"
)

// ProviderJSON represents the complete provider details from the JSON file.
type ProviderJSON struct {
	Profile struct {
		ProviderName     string  `json:"providerName"`
		ProviderType     string  `json:"providerType"`
		Email            string  `json:"email"`
		PhoneNumber      string  `json:"phoneNumber"`
		Status           string  `json:"status"`
		AdvancedVerified bool    `json:"advancedVerified"`
		ProfileImage     string  `json:"profileImage"`
		Address          string  `json:"address"`
		Rating           float64 `json:"rating"`
		LocationGeo      struct {
			Type        string    `json:"type"`
			Coordinates []float64 `json:"coordinates"`
		} `json:"locationGeo"`
	} `json:"profile"`
	Security struct {
		Password string `json:"password"` // Should be "$Muguchia1"
	} `json:"security"`
	ServiceCatalogue struct {
		ServiceType   string                 `json:"serviceType"`
		Mode          string                 `json:"mode"`
		CustomOptions map[string]interface{} `json:"customOptions"`
	} `json:"serviceCatalogue"`
	Verification struct {
		KYPDocument        string `json:"kypDocument"`
		VerificationStatus string `json:"verificationStatus"`
		LegalName          string `json:"legalName"`
		VerificationCode   string `json:"verificationCode"`
	} `json:"verification"`
}

// ProviderRegistrationRequest represents the payload for each registration step.
type ProviderRegistrationRequest struct {
	Step             string                 `json:"step"`
	SessionID        string                 `json:"sessionID,omitempty"`
	OTP              string                 `json:"otp,omitempty"`
	BasicData        map[string]interface{} `json:"basicData,omitempty"`
	KYPData          map[string]interface{} `json:"kypData,omitempty"`
	ServiceCatalogue map[string]interface{} `json:"serviceCatalogue,omitempty"`
}

const registrationURL = "http://192.168.100.19:8080/api/providers/register"

var httpClient = &http.Client{Timeout: 10 * time.Second}

// postJSON sends a POST request with the provided JSON payload and custom headers.
func postJSON(url string, payload interface{}, headers map[string]string) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("json marshal failed: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, val := range headers {
		req.Header.Set(key, val)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return result, nil
}

func getOTP(sessionID string) (string, error) {
	testKey := fmt.Sprintf("session:%s", sessionID)
	ctx := context.Background()
	client := utils.GetTestCacheClient()
	// Poll for up to 10 seconds.
	for i := 0; i < 10; i++ {
		otp, err := client.Get(ctx, testKey).Result()
		if err == nil && otp != "" {
			return otp, nil
		}
		time.Sleep(1 * time.Second)
	}
	return "", fmt.Errorf("OTP not found for key %s", testKey)
}

func registerProvider(prov ProviderJSON, deviceID string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Introduce a random delay (0-500ms) to reduce simultaneous collisions.
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

	// Device headers for this provider registration.
	deviceHeaders := map[string]string{
		"X-Device-ID":   deviceID,
		"X-Device-Name": "SimDevice_" + deviceID,
	}

	log.Printf("[%s] Starting registration...", prov.Profile.ProviderName)

	// STEP 1: Basic Registration.
	basicData := map[string]interface{}{
		"providerName": prov.Profile.ProviderName,
		"providerType": prov.Profile.ProviderType,
		"email":        prov.Profile.Email,
		"phoneNumber":  prov.Profile.PhoneNumber,
		"password":     prov.Security.Password,
		"address":      prov.Profile.Address,
		"profileImage": prov.Profile.ProfileImage,
		"rating":       prov.Profile.Rating,
		"locationGeo":  prov.Profile.LocationGeo,
	}
	basicReq := ProviderRegistrationRequest{
		Step:      "basic",
		BasicData: basicData,
	}
	log.Printf("[%s] Basic registration payload: %+v", prov.Profile.ProviderName, basicData)
	basicResp, err := postJSON(registrationURL, basicReq, deviceHeaders)
	if err != nil {
		log.Printf("[%s] Basic registration failed: %v", prov.Profile.ProviderName, err)
		return
	}
	sessionID, ok := basicResp["sessionID"].(string)
	if !ok || sessionID == "" {
		log.Printf("[%s] No sessionID returned. Response: %+v", prov.Profile.ProviderName, basicResp)
		return
	}
	log.Printf("[%s] Received sessionID: %s", prov.Profile.ProviderName, sessionID)

	// STEP 2: Retrieve OTP from Redis.
	log.Printf("[%s] Waiting for OTP...", prov.Profile.ProviderName)
	otp, err := getOTP(sessionID)
	if err != nil {
		log.Printf("[%s] OTP retrieval failed: %v", prov.Profile.ProviderName, err)
		return
	}
	log.Printf("[%s] Retrieved OTP: %s", prov.Profile.ProviderName, otp)

	// STEP 3: OTP Verification.
	otpReq := ProviderRegistrationRequest{
		Step:      "otp",
		SessionID: sessionID,
		OTP:       otp,
	}
	otpResp, err := postJSON(registrationURL, otpReq, deviceHeaders)
	if err != nil {
		log.Printf("[%s] OTP verification failed: %v", prov.Profile.ProviderName, err)
		return
	}
	log.Printf("[%s] OTP verification response: %+v", prov.Profile.ProviderName, otpResp)

	// STEP 4: KYP Verification.
	kypData := map[string]interface{}{
		"documentURL": prov.Verification.KYPDocument,
		"legalName":   prov.Verification.LegalName,
		"selfieURL":   fmt.Sprintf("http://example.com/selfie/%s.jpg", prov.Profile.ProviderName),
	}
	kypReq := ProviderRegistrationRequest{
		Step:      "kyp",
		SessionID: sessionID,
		KYPData:   kypData,
	}
	kypResp, err := postJSON(registrationURL, kypReq, deviceHeaders)
	if err != nil {
		log.Printf("[%s] KYP verification failed: %v", prov.Profile.ProviderName, err)
		return
	}
	log.Printf("[%s] KYP verification response: %+v", prov.Profile.ProviderName, kypResp)

	// STEP 5: Finalize Registration with Service Catalogue.
	serviceCatalogue := map[string]interface{}{
		"serviceType":   prov.ServiceCatalogue.ServiceType,
		"mode":          prov.ServiceCatalogue.Mode,
		"customOptions": prov.ServiceCatalogue.CustomOptions,
	}
	catalogueReq := ProviderRegistrationRequest{
		Step:             "catalogue",
		SessionID:        sessionID,
		ServiceCatalogue: serviceCatalogue,
	}
	catalogueResp, err := postJSON(registrationURL, catalogueReq, deviceHeaders)
	if err != nil {
		log.Printf("[%s] Finalization failed: %v", prov.Profile.ProviderName, err)
		return
	}
	log.Printf("[%s] Final registration response: %+v", prov.Profile.ProviderName, catalogueResp)
}

func test() {
	rand.Seed(time.Now().UnixNano())

	// Open and read the providers JSON file.
	filePath := "providers.json"
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read JSON file: %v", err)
	}

	var providers []ProviderJSON
	if err := json.Unmarshal(fileData, &providers); err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	log.Printf("Found %d providers in the JSON file.", len(providers))

	// Use a WaitGroup to spawn a goroutine for each provider.
	var wg sync.WaitGroup
	for i, prov := range providers {
		wg.Add(1)
		deviceID := fmt.Sprintf("device_%d", i+1)
		go registerProvider(prov, deviceID, &wg)
	}

	wg.Wait()
	log.Println("All provider registrations complete.")
}
