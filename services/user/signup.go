package user

import (
	"fmt"
	"time"

	"bloomify/models"
	"bloomify/utils"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// InitiateRegistration validates basic data, checks for duplicates, creates a registration session,
// initiates OTP, and returns the session ID with code 100 (OTP pending).
func (s *DefaultUserService) InitiateRegistration(basicReq models.UserBasicRegistrationData, device models.Device) (string, int, error) {
	if basicReq.Email == "" || basicReq.Password == "" || basicReq.Username == "" || basicReq.PhoneNumber == "" {
		return "", 0, fmt.Errorf("all fields are required")
	}

	// Check if username/email is available
	available, err := s.Repo.IsUserAvailable(basicReq)
	if err != nil {
		utils.GetLogger().Error("InitiateRegistration: availability check failed", zap.Error(err))
		return "", 0, fmt.Errorf("registration failed, please try again")
	}
	if !available {
		return "", 0, fmt.Errorf("a user with this email or username already exists")
	}

	existing, err := s.Repo.GetByEmailWithProjection(basicReq.Email, bson.M{"id": 1})
	if err != nil {
		utils.GetLogger().Error("InitiateRegistration: failed to check for existing user", zap.Error(err))
		return "", 0, fmt.Errorf("registration failed, please try again")
	}
	if existing != nil {
		return "", 0, fmt.Errorf("a user with this email already exists")
	}

	sessionClient := utils.GetAuthCacheClient()
	sessionID := fmt.Sprintf("%s:%s", basicReq.Email, device.DeviceID)

	regSession := models.UserRegistrationSession{
		TempID: sessionID,
		BasicData: &models.UserBasicRegistrationData{
			Username:    basicReq.Username,
			Email:       basicReq.Email,
			Password:    basicReq.Password,
			PhoneNumber: basicReq.PhoneNumber,
		},
		OTPStatus:     "pending",
		CreatedAt:     time.Now(),
		LastUpdatedAt: time.Now(),
		Devices:       []models.Device{device},
	}

	if err := utils.InitiateDeviceOTP(basicReq.Email, device.DeviceID, basicReq.PhoneNumber); err != nil {
		return "", 0, fmt.Errorf("failed to initiate OTP: %w", err)
	}

	if err := SaveUserRegistrationSession(sessionClient, sessionID, regSession, 30*time.Minute); err != nil {
		return "", 0, fmt.Errorf("failed to save registration session: %w", err)
	}

	// Return sessionID with code 100 (OTP pending).
	return sessionID, 100, nil
}

// VerifyRegistrationOTP retrieves the session, verifies the OTP, updates the session to "verified",
// and returns code 101 (OTP verified).
func (s *DefaultUserService) VerifyRegistrationOTP(sessionID string, deviceID string, providedOTP string) (int, error) {
	sessionClient := utils.GetAuthCacheClient()
	regSession, err := GetUserRegistrationSession(sessionClient, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve registration session")
	}

	if err := utils.VerifyDeviceOTPRecord(regSession.BasicData.Email, deviceID, providedOTP); err != nil {
		return 0, fmt.Errorf("OTP verification failed: %w", err)
	}

	regSession.OTPStatus = "verified"
	regSession.LastUpdatedAt = time.Now()
	if err := SaveUserRegistrationSession(sessionClient, sessionID, regSession, 30*time.Minute); err != nil {
		return 0, fmt.Errorf("failed to update registration session: %w", err)
	}

	// Return code 101 to indicate OTP verified.
	return 101, nil
}

// FinalizeRegistration retrieves the session, builds and persists the user record using stored basic data and provided preferences,
// clears the registration session, and returns an AuthResponse. (Finalization corresponds to code 102.)
func (s *DefaultUserService) FinalizeRegistration(sessionID string, preferences []string) (*AuthResponse, error) {
	sessionClient := utils.GetAuthCacheClient()
	regSession, err := GetUserRegistrationSession(sessionClient, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve registration session")
	}
	if regSession.OTPStatus != "verified" {
		return nil, fmt.Errorf("OTP not verified")
	}
	if regSession.BasicData == nil {
		return nil, fmt.Errorf("registration session missing basic data")
	}

	if err := VerifyPasswordComplexity(regSession.BasicData.Password); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(regSession.BasicData.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.GetLogger().Error("FinalizeRegistration: Failed to hash password", zap.Error(err))
		return nil, fmt.Errorf("registration failed, please try again")
	}

	userObj := models.User{
		Username:     regSession.BasicData.Username,
		Email:        regSession.BasicData.Email,
		PhoneNumber:  regSession.BasicData.PhoneNumber,
		PasswordHash: string(hashedPassword),
		Password:     "",
		Preferences:  preferences,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	userObj.ID = uuid.New().String()

	if len(regSession.Devices) == 0 {
		return nil, fmt.Errorf("registration session missing device information")
	}
	device := regSession.Devices[0]
	now := time.Now()
	device.LastLogin = now
	device.Creator = true

	token, err := utils.GenerateToken(userObj.ID, userObj.Email, device.DeviceID)
	if err != nil {
		utils.GetLogger().Error("FinalizeRegistration: Failed to generate auth token", zap.Error(err))
		return nil, fmt.Errorf("registration failed, please try again")
	}
	tokenHash := utils.HashToken(token)
	device.TokenHash = tokenHash

	userObj.Devices = []models.Device{device}

	if err := s.Repo.Create(&userObj); err != nil {
		utils.GetLogger().Error("FinalizeRegistration: Failed to create user", zap.Error(err))
		return nil, fmt.Errorf("registration failed, please try again")
	}

	_ = DeleteUserRegistrationSession(sessionClient, sessionID)

	return &AuthResponse{
		ID:           userObj.ID,
		Token:        token,
		Username:     userObj.Username,
		Email:        userObj.Email,
		PhoneNumber:  userObj.PhoneNumber,
		ProfileImage: userObj.ProfileImage,
		Rating:       userObj.Rating,
	}, nil
}
