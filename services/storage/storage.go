package storage

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/cloudinary/cloudinary-go/v2/asset"
)

// NewStorageService creates a new StorageServiceImpl instance.
func NewStorageService(cld *cloudinary.Cloudinary, cloudName, apiSecret string) StorageService {
	fmt.Printf("[DEBUG] Initializing StorageServiceImpl with cloudName: %s\n", cloudName)
	return &StorageServiceImpl{
		cld:       cld,
		cloudName: cloudName,
		apiSecret: apiSecret,
	}
}

// UploadFile uploads a file to Cloudinary into the specified folder and returns the permanent identifier.
func (s *StorageServiceImpl) UploadFile(ctx context.Context, localFilePath, destFolder string) (string, error) {
	uploadParams := uploader.UploadParams{
		Folder: destFolder,
	}
	result, err := s.cld.Upload.Upload(ctx, localFilePath, uploadParams)
	if err != nil {
		return "", fmt.Errorf("StorageServiceImpl: failed to upload file: %w", err)
	}
	if result.PublicID == "" {
		return "", fmt.Errorf("StorageServiceImpl: no public ID returned")
	}
	return result.PublicID, nil
}

// DeleteFile deletes a file from Cloudinary given its public ID.
func (s *StorageServiceImpl) DeleteFile(ctx context.Context, publicID string) error {
	_, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: publicID})
	if err != nil {
		return fmt.Errorf("StorageServiceImpl: failed to delete file: %w", err)
	}
	return nil
}

// getAsset returns an asset instance based on the resource type.
func (s *StorageServiceImpl) getAsset(resourceType, publicID string) (*asset.Asset, error) {
	switch resourceType {
	case "image":
		return s.cld.Image(publicID)
	case "video":
		return s.cld.Video(publicID)
	default:
		return s.cld.Media(publicID)
	}
}

// GetDownloadURL constructs a public URL for a file based on its resource type.
func (s *StorageServiceImpl) GetDownloadURL(ctx context.Context, resourceType, publicID string, expires time.Duration) (string, error) {
	a, err := s.getAsset(resourceType, publicID)
	if err != nil {
		return "", fmt.Errorf("StorageServiceImpl: failed to get asset: %w", err)
	}
	url, err := a.String()
	if err != nil {
		return "", fmt.Errorf("StorageServiceImpl: failed to get URL string: %w", err)
	}
	return url, nil
}

// GetSecureDownloadURL generates a signed, short-lived URL for an authenticated resource.
// It manually computes a signature using SHA-1 over "expires_at" and "public_id" concatenated with the API secret.
func (s *StorageServiceImpl) GetSecureDownloadURL(ctx context.Context, resourceType, publicID string, expires time.Duration) (string, error) {
	expiresAt := time.Now().Add(expires).Unix()
	// Build the string to sign.
	stringToSign := fmt.Sprintf("expires_at=%d&public_id=%s%s", expiresAt, publicID, s.apiSecret)
	signature := computeSHA1(stringToSign)
	// Construct the secure URL manually.
	secureURL := fmt.Sprintf("https://res.cloudinary.com/%s/%s/authenticated/s--%s--/expires_%d/%s", s.cloudName, resourceType, signature, expiresAt, publicID)
	return secureURL, nil
}

// computeSHA1 computes the SHA-1 hash of the input and returns its hex encoding.
func computeSHA1(input string) string {
	h := sha1.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// UploadKYPFile encrypts the file and uploads it for KYP purposes.
// It returns the permanent file identifier (e.g., Cloudinary PublicID).
func (s *StorageServiceImpl) UploadKYPFile(ctx context.Context, localFilePath, destFolder, adminKey string) (string, error) {
	encryptedFilePath, err := encryptFile(localFilePath, adminKey)
	if err != nil {
		return "", fmt.Errorf("StorageServiceImpl: failed to encrypt file: %w", err)
	}
	publicID, err := s.UploadFile(ctx, encryptedFilePath, destFolder)
	if err != nil {
		return "", fmt.Errorf("StorageServiceImpl: failed to upload encrypted KYP file: %w", err)
	}
	return publicID, nil
}
