package storage

import (
	"bloomify/config"
	"bloomify/utils"
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// StorageService defines the interface for storage operations.
type StorageService interface {
	UploadFile(ctx context.Context, localFilePath, destFolder string) (string, error)
	DeleteFile(ctx context.Context, publicID string) error
	GetDownloadURL(ctx context.Context, publicID string, expires time.Duration) (string, error)
	GetSecureDownloadURL(ctx context.Context, publicID string, expires time.Duration) (string, error)
	UploadEncryptedFile(ctx context.Context, localFilePath, destFolder, encryptionKey string) (string, error)
}

// FirebaseStorageService implements StorageService using Firebase Storage.
type FirebaseStorageService struct {
	client         *storage.Client
	bucketName     string
	serviceAccount *config.ServiceAccount
}

// NewFirebaseStorageService creates a new FirebaseStorageService.
func NewFirebaseStorageService(serviceAccountJSONPath, bucketName string) (*FirebaseStorageService, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(serviceAccountJSONPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	// Load service account for signing URLs
	sa, err := utils.LoadServiceAccount(serviceAccountJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load service account for signing URLs: %w", err)
	}

	return &FirebaseStorageService{
		client:         client,
		bucketName:     bucketName,
		serviceAccount: sa,
	}, nil
}

// UploadFile needs proper content type handling
func (s *FirebaseStorageService) UploadFile(ctx context.Context, localFilePath, destFolder string) (string, error) {
	file, err := os.Open(localFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	objectPath := filepath.Join(destFolder, filepath.Base(localFilePath))
	obj := s.client.Bucket(s.bucketName).Object(objectPath)
	w := obj.NewWriter(ctx)

	// Set public read ACL
	w.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}

	// Detect and set content type
	if ext := filepath.Ext(localFilePath); ext != "" {
		w.ObjectAttrs.ContentType = mime.TypeByExtension(ext)
	}

	if _, err := io.Copy(w, file); err != nil {
		return "", fmt.Errorf("failed to copy file to storage: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	return objectPath, nil
}

// DeleteFile deletes an object from the bucket.
func (s *FirebaseStorageService) DeleteFile(ctx context.Context, publicID string) error {
	obj := s.client.Bucket(s.bucketName).Object(publicID)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetDownloadURL returns a public URL assuming the file is publicly accessible.
func (s *FirebaseStorageService) GetDownloadURL(ctx context.Context, publicID string, expires time.Duration) (string, error) {
	// public URL format (no signing)
	url := fmt.Sprintf("https://firebasestorage.googleapis.com/v0/b/%s/o/%s?alt=media", s.bucketName, urlEncode(publicID))
	return url, nil
}

// GetSecureDownloadURL returns a signed URL valid for the specified duration.
func (s *FirebaseStorageService) GetSecureDownloadURL(ctx context.Context, publicID string, expires time.Duration) (string, error) {
	url, err := storage.SignedURL(s.bucketName, publicID, &storage.SignedURLOptions{
		GoogleAccessID: s.serviceAccount.ClientEmail,
		PrivateKey:     []byte(strings.ReplaceAll(s.serviceAccount.PrivateKey, `\n`, "\n")),
		Method:         "GET",
		Expires:        time.Now().Add(expires),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}
	return url, nil
}

// UploadEncryptedFile encrypts and uploads a file using AES-256 GCM.
func (s *FirebaseStorageService) UploadEncryptedFile(ctx context.Context, localFilePath, destFolder, encryptionKey string) (string, error) {
	encryptedPath, err := encryptFile(localFilePath, encryptionKey)
	if err != nil {
		return "", err
	}
	defer os.Remove(encryptedPath)

	data, err := os.ReadFile(encryptedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read encrypted file: %w", err)
	}

	objectPath := filepath.Join(destFolder, filepath.Base(localFilePath))
	obj := s.client.Bucket(s.bucketName).Object(objectPath)
	w := obj.NewWriter(ctx)

	if _, err := w.Write(data); err != nil {
		return "", fmt.Errorf("failed to write encrypted data: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	return objectPath, nil
}

// UploadWithContext uploads a file handling folder structure and encryption based on parameters.
// isPrivate: whether file is private (encrypted)
// isKYP: whether the file is related to KYP (private KYP)
// isImage: whether the file is an image (vs document/file)
// encryptionKey: required if isPrivate==true
func (s *FirebaseStorageService) UploadWithContext(ctx context.Context, localFilePath string, isPrivate, isKYP, isImage bool, encryptionKey string) (string, error) {
	var folder string

	if isPrivate {
		if isKYP {
			if isImage {
				folder = "private/kyp/images"
			} else {
				folder = "private/kyp/files"
			}
		} else {
			if isImage {
				folder = "private/images"
			} else {
				folder = "private/files"
			}
		}
		return s.UploadEncryptedFile(ctx, localFilePath, folder, encryptionKey)
	}

	// Public uploads
	if isImage {
		folder = "public/images"
	} else {
		folder = "public/files"
	}
	return s.UploadFile(ctx, localFilePath, folder)
}

func urlEncode(s string) string {
	return url.QueryEscape(s)
}
