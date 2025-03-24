package storage

import (
	"context"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
)

// StorageService defines operations for file storage.
type StorageService interface {
	UploadFile(ctx context.Context, localFilePath, destFolder string) (string, error)
	DeleteFile(ctx context.Context, publicID string) error
	// GetDownloadURL constructs a public URL for a file given its resource type and permanent identifier.
	GetDownloadURL(ctx context.Context, resourceType, publicID string, expires time.Duration) (string, error)
	// GetSecureDownloadURL generates a signed, short-lived URL for an authenticated resource.
	GetSecureDownloadURL(ctx context.Context, resourceType, publicID string, expires time.Duration) (string, error)
	// UploadKYPFile encrypts and uploads a KYP file using the provided adminKey.
	// It returns the permanent file identifier (e.g., Cloudinary PublicID).
	UploadKYPFile(ctx context.Context, localFilePath, destFolder, adminKey string) (string, error)
}

// StorageServiceImpl implements StorageService using Cloudinary.
type StorageServiceImpl struct {
	cld       *cloudinary.Cloudinary
	cloudName string
	apiSecret string
}
