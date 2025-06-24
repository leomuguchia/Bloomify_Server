// File: provider/service.go
package provider

import (
	"fmt"
	"strings"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func (s *DefaultProviderService) RegisterBasic(basicReq models.ProviderBasicRegistrationData, device models.Device) (string, int, error) {
	basicReq.Email = strings.ToLower(basicReq.Email)
	if err := validateBasicRegistrationData(basicReq); err != nil {
		return "", 0, fmt.Errorf("validation error: %w", err)
	}

	available, err := s.Repo.IsProviderAvailable(basicReq)
	if err != nil {
		return "", 0, fmt.Errorf("availability check failed: %w", err)
	}
	if available {
		return "", 0, fmt.Errorf("a provider with this email or username already exists")
	}

	sessionID := GenerateSessionID()
	now := time.Now()

	if err := utils.InitiateDeviceOTP(sessionID, device.DeviceID, basicReq.PhoneNumber); err != nil {
		return "", 0, fmt.Errorf("failed to initiate OTP: %w", err)
	}

	session := models.ProviderRegistrationSession{
		TempID:        sessionID,
		BasicData:     basicReq,
		OTPStatus:     "pending",
		CreatedAt:     now,
		LastUpdatedAt: now,
		Devices:       []models.Device{device},
	}

	authCacheClient := utils.GetProviderAuthCacheClient()
	if err := SaveRegistrationSession(authCacheClient, sessionID, session, 30*time.Minute); err != nil {
		return "", 0, fmt.Errorf("failed to save registration session: %w", err)
	}

	return sessionID, 100, nil
}

// VerifyOTP verifies the OTP for registration.
// It retrieves the session, validates the OTP using sessionID and deviceID,
// updates the session's OTP status upon success, and returns status 105.
func (s *DefaultProviderService) VerifyOTP(sessionID string, deviceID string, providedOTP string) (int, error) {
	authCacheClient := utils.GetProviderAuthCacheClient()

	session, err := GetRegistrationSession(authCacheClient, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve registration session")
	}

	if err := utils.VerifyDeviceOTPRecord(sessionID, deviceID, providedOTP); err != nil {
		return 0, fmt.Errorf("OTP verification failed: %w", err)
	}

	session.OTPStatus = "verified"
	session.LastUpdatedAt = time.Now()
	if err := SaveRegistrationSession(authCacheClient, sessionID, session, 30*time.Minute); err != nil {
		return 0, fmt.Errorf("failed to update OTP status in registration session: %w", err)
	}

	return 105, nil
}

// VerifyKYP handles Step 2: KYP Verification.
// It retrieves the registration session and updates it with the provided KYP details.
func (s *DefaultProviderService) VerifyKYP(sessionID string, kypData models.KYPVerificationData) (int, error) {
	authCacheClient := utils.GetProviderAuthCacheClient()

	session, err := GetRegistrationSession(authCacheClient, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve registration session")
	}

	switch kypData.Type {
	case "freelancer":
		if kypData.DocumentURL == "" || kypData.LegalName == "" || kypData.SelfieURL == "" {
			return 10, fmt.Errorf("missing verification details for freelancer")
		}
	case "business":
		if kypData.DocumentURL == "" || kypData.LegalName == "" || kypData.ContactName == "" || kypData.ContactEmail == "" || kypData.DocumentType == "" {
			return 10, fmt.Errorf("missing verification details for business")
		}
	default:
		return 10, fmt.Errorf("invalid provider type: %s", kypData.Type)
	}

	session.KYPData = kypData
	session.VerificationStatus = "verified"
	session.VerificationLevel = "basic"
	session.LastUpdatedAt = time.Now()

	if err := SaveRegistrationSession(authCacheClient, sessionID, session, 30*time.Minute); err != nil {
		return 0, fmt.Errorf("failed to update registration session: %w", err)
	}

	return 101, nil
}

// FinalizeRegistration handles Step 3: Service Catalogue and full persistence.
// It retrieves the registration session, updates it with service catalogue details,
// converts it into a full Provider model, generates a JWT token (using your utils functions),
// updates the device's token hash, persists the Provider record, and clears the session.
func (s *DefaultProviderService) FinalizeRegistration(sessionID string, catalogueData models.ServiceCatalogue) (*models.ProviderAuthResponse, error) {
	authCacheClient := utils.GetProviderAuthCacheClient()

	session, err := GetRegistrationSession(authCacheClient, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve registration session: %w", err)
	}

	if catalogueData.Service.ID == "" || catalogueData.Mode == "" {
		return nil, fmt.Errorf("service type and mode are required")
	}

	// Update session with the service catalogue details.
	session.ServiceCatalogue = catalogueData
	session.LastUpdatedAt = time.Now()

	// Directly build the Provider model from the session.
	provider := models.Provider{
		ID: GenerateProviderID(),
		Profile: models.Profile{
			ProviderName: session.BasicData.ProviderName,
			ProviderType: session.BasicData.ProviderType,
			Email:        session.BasicData.Email,
			PhoneNumber:  session.BasicData.PhoneNumber,
			Address:      session.BasicData.Address,
			Status:       "new",
			ProfileImage: session.BasicData.ProfileImageUrl,
			LocationGeo:  session.BasicData.LocationGeo,
			Rating:       3.0,
			Description:  session.BasicData.Description,
		},
		ServiceCatalogue: session.ServiceCatalogue,
		BasicVerification: models.BasicVerification{
			LegalName:          session.KYPData.LegalName,
			KYPDocument:        session.KYPData.DocumentURL,
			VerificationStatus: session.VerificationStatus,
		},
		VerificationLevel: "basic",
		Devices:           session.Devices,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Hash the password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(session.BasicData.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	provider.Security.PasswordHash = string(hashedPassword)
	provider.Security.Password = ""

	// Generate a token and update the device's token hash.
	registrationDevice := session.Devices[0]
	token, err := utils.GenerateToken(provider.ID, provider.Profile.Email, registrationDevice.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}
	tokenHash := utils.HashToken(token)
	deviceUpdated := false
	for idx, d := range provider.Devices {
		if d.DeviceID == registrationDevice.DeviceID {
			provider.Devices[idx].TokenHash = tokenHash
			provider.Devices[idx].LastLogin = time.Now()
			deviceUpdated = true
			break
		}
	}
	if !deviceUpdated {
		registrationDevice.TokenHash = tokenHash
		registrationDevice.LastLogin = time.Now()
		provider.Devices = append(provider.Devices, registrationDevice)
	}

	// Persist the provider record.
	if err := s.Repo.Create(&provider); err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Delete the registration session.
	if err := DeleteRegistrationSession(authCacheClient, sessionID); err != nil {
		utils.GetLogger().Error("Failed to delete registration session", zap.String("sessionID", sessionID), zap.Error(err))
	}

	// Build and return the authentication response.
	resp := &models.ProviderAuthResponse{
		ID:               provider.ID,
		Token:            token,
		Profile:          provider.Profile,
		CreatedAt:        provider.CreatedAt,
		ServiceCatalogue: provider.ServiceCatalogue,
	}
	return resp, nil
}
