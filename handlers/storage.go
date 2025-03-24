package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bloomify/services/storage"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// StorageHandler handles both general file and KYP file storage endpoints.
type StorageHandler struct {
	StorageSvc storage.StorageService
	AdminKey   string
}

// NewStorageHandler creates a new StorageHandler instance.
// It now fetches the adminKey from configuration.
func NewStorageHandler(svc storage.StorageService) *StorageHandler {
	adminKey := viper.GetString("cloudinary.adminKey")
	return &StorageHandler{
		StorageSvc: svc,
		AdminKey:   adminKey,
	}
}

// allowedBuckets defines permitted buckets for general file uploads.
var allowedBuckets = map[string]bool{
	"images": true,
	"videos": true,
}

// allowedKYPBuckets defines permitted buckets for KYP files.
var allowedKYPBuckets = map[string]bool{
	"documents": true,
	"selfies":   true,
}

// UploadFileHandler handles general file uploads.
func (h *StorageHandler) UploadFileHandler(c *gin.Context) {
	fileType := c.Param("type")
	bucket := c.Param("bucket")
	if !allowedBuckets[bucket] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket; allowed values are 'images' and 'videos'"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not provided", "detail": err.Error()})
		return
	}

	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, fileHeader.Filename)
	if err := c.SaveUploadedFile(fileHeader, tempFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file", "detail": err.Error()})
		return
	}
	defer os.Remove(tempFilePath)

	destFolder := fileType + "s/" + bucket

	publicID, err := h.StorageSvc.UploadFile(c, tempFilePath, destFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file", "detail": err.Error()})
		return
	}

	downloadURL, err := h.StorageSvc.GetDownloadURL(c, fileType, publicID, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to construct download URL", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "file uploaded successfully",
		"downloadURL": downloadURL,
	})
}

// GetDownloadURLHandler generates a public download URL for general files.
func (h *StorageHandler) GetDownloadURLHandler(c *gin.Context) {
	fileType := c.Param("type")
	bucket := c.Param("bucket")
	filename := c.Param("filename")
	if !allowedBuckets[bucket] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket; allowed values are 'images' and 'videos'"})
		return
	}

	destPath := fileType + "s/" + bucket + "/" + filename

	expiry := 15 * time.Minute
	if expStr := c.Query("expires"); expStr != "" {
		if exp, err := time.ParseDuration(expStr); err == nil {
			expiry = exp
		}
	}

	url, err := h.StorageSvc.GetDownloadURL(c, fileType, destPath, expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download URL", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"downloadURL": url})
}

// KYPUploadFileHandler handles KYP file uploads (documents and selfies).
func (h *StorageHandler) KYPUploadFileHandler(c *gin.Context) {
	bucket := c.Param("bucket")
	if !allowedKYPBuckets[bucket] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket; allowed values are 'documents' and 'selfies'"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not provided", "detail": err.Error()})
		return
	}

	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, fileHeader.Filename)
	if err := c.SaveUploadedFile(fileHeader, tempFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file", "detail": err.Error()})
		return
	}
	defer os.Remove(tempFilePath)

	destFolder := "kyp/" + bucket

	publicID, err := h.StorageSvc.UploadKYPFile(c, tempFilePath, destFolder, h.AdminKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload KYP file", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "KYP file uploaded successfully",
		"permanentFileID": publicID,
	})
}

func (h *StorageHandler) KYPGetDownloadURLHandler(c *gin.Context) {
	adminToken, exists := c.Get("adminToken")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "admin token not found"})
		return
	}

	expectedKey := viper.GetString("cloudinary.adminKey")
	if tokenStr, ok := adminToken.(string); !ok || tokenStr != expectedKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin key"})
		return
	}

	bucket := c.Param("bucket")
	filename := c.Param("filename")
	if !allowedKYPBuckets[bucket] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket; allowed values are 'documents' and 'selfies'"})
		return
	}

	destPath := "kyp/" + bucket + "/" + filename

	expiry := 15 * time.Minute
	if expStr := c.Query("expires"); expStr != "" {
		if exp, err := time.ParseDuration(expStr); err == nil {
			expiry = exp
		}
	}

	secureURL, err := h.StorageSvc.GetSecureDownloadURL(c, bucket, destPath, expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate secure download URL", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"downloadURL": secureURL})
}
