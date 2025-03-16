package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudinary/cloudinary-go"
	"github.com/cloudinary/cloudinary-go/api/uploader"
)

// StorageService defines operations for file storage.
type StorageService interface {
	// UploadFile uploads a file from localFilePath into the specified folder (destFolder)
	// and returns the secure URL.
	UploadFile(ctx context.Context, localFilePath, destFolder string) (string, error)
	// DeleteFile deletes a file from Cloudinary given its public ID.
	DeleteFile(ctx context.Context, destPath string) error
	// GetDownloadURL constructs a secure URL for a file given its destination path.
	GetDownloadURL(ctx context.Context, destPath string, expires time.Duration) (string, error)
}

// StorageServiceImpl implements StorageService using Cloudinary.
type StorageServiceImpl struct {
	cld       *cloudinary.Cloudinary
	cloudName string
}

// NewStorageService creates a new StorageServiceImpl instance.
func NewStorageService(cld *cloudinary.Cloudinary, cloudName string) StorageService {
	fmt.Printf("[DEBUG] Initializing StorageServiceImpl with cloudName: %s\n", cloudName)
	return &StorageServiceImpl{
		cld:       cld,
		cloudName: cloudName,
	}
}

// UploadFile uploads a file to Cloudinary into the specified folder and returns the secure URL.
func (s *StorageServiceImpl) UploadFile(ctx context.Context, localFilePath, destFolder string) (string, error) {
	fmt.Printf("[DEBUG] Starting upload. Local file path: %s, Destination folder: %s\n", localFilePath, destFolder)

	uploadParams := uploader.UploadParams{
		Folder: destFolder,
	}
	fmt.Printf("[DEBUG] Upload parameters: %+v\n", uploadParams)

	result, err := s.cld.Upload.Upload(ctx, localFilePath, uploadParams)
	if err != nil {
		fmt.Printf("[ERROR] Upload failed: %v\n", err)
		return "", fmt.Errorf("StorageServiceImpl: failed to upload file: %w", err)
	}

	// Log detailed response information for debugging.
	fmt.Printf("[DEBUG] Cloudinary Upload Response:\n")
	fmt.Printf("  PublicID: %s\n", result.PublicID)
	fmt.Printf("  SecureURL: %s\n", result.SecureURL)

	if result.SecureURL == "" {
		return "", fmt.Errorf("StorageServiceImpl: secure URL is empty in upload response")
	}
	return result.SecureURL, nil
}

// DeleteFile deletes a file from Cloudinary given its public ID.
func (s *StorageServiceImpl) DeleteFile(ctx context.Context, destPath string) error {
	fmt.Printf("[DEBUG] Deleting file with PublicID: %s\n", destPath)
	_, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: destPath})
	if err != nil {
		fmt.Printf("[ERROR] Delete failed: %v\n", err)
		return fmt.Errorf("StorageServiceImpl: failed to delete file: %w", err)
	}
	fmt.Println("[DEBUG] File deletion successful.")
	return nil
}

// GetDownloadURL constructs a secure URL for a file in Cloudinary.
// Cloudinary secure URLs are permanent by default.
func (s *StorageServiceImpl) GetDownloadURL(ctx context.Context, destPath string, expires time.Duration) (string, error) {
	url := fmt.Sprintf("https://res.cloudinary.com/%s/image/upload/%s", s.cloudName, destPath)
	fmt.Printf("[DEBUG] Constructed download URL: %s\n", url)
	if url == "" {
		return "", fmt.Errorf("StorageServiceImpl: failed to construct download URL")
	}
	return url, nil
}
