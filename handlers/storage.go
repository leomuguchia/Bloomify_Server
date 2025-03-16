package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bloomify/services/storage"

	"github.com/gin-gonic/gin"
)

// StorageHandler handles file storage HTTP endpoints.
type StorageHandler struct {
	StorageSvc storage.StorageService
}

// NewStorageHandler creates a new StorageHandler.
func NewStorageHandler(svc storage.StorageService) *StorageHandler {
	return &StorageHandler{
		StorageSvc: svc,
	}
}

// allowedBuckets defines the permitted buckets.
var allowedBuckets = map[string]bool{
	"images": true,
	"videos": true,
}

// UploadFileHandler handles file uploads.
// URL pattern: POST /storage/:type/:bucket/upload
// where :type is "user" or "provider" and :bucket is "images" or "videos".
// The file is expected as multipart/form-data with the key "file".
func (h *StorageHandler) UploadFileHandler(c *gin.Context) {
	fileType := c.Param("type") // "user" or "provider"
	bucket := c.Param("bucket") // must be "images" or "videos"

	// Ensure bucket is allowed.
	if !allowedBuckets[bucket] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket, allowed values are 'images' and 'videos'"})
		return
	}

	// Retrieve the uploaded file.
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not provided", "detail": err.Error()})
		return
	}

	// Save the file to a temporary location.
	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, fileHeader.Filename)
	if err := c.SaveUploadedFile(fileHeader, tempFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file", "detail": err.Error()})
		return
	}
	// Clean up the temporary file after processing.
	defer os.Remove(tempFilePath)

	// Construct destination folder.
	destFolder := fileType + "s/" + bucket
	// Optionally, you might use the same filename as a public ID.
	// In our implementation, UploadFile returns a secure URL directly.
	secureURL, err := h.StorageSvc.UploadFile(c, tempFilePath, destFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "file uploaded successfully",
		"downloadURL": secureURL,
	})
}

// GetDownloadURLHandler generates a signed URL for a file.
// URL pattern: GET /storage/:type/:bucket/:filename?expires=15m
func (h *StorageHandler) GetDownloadURLHandler(c *gin.Context) {
	fileType := c.Param("type") // "user" or "provider"
	bucket := c.Param("bucket") // "images" or "videos"
	filename := c.Param("filename")

	// Ensure bucket is allowed.
	if !allowedBuckets[bucket] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket, allowed values are 'images' and 'videos'"})
		return
	}

	// Construct destination path.
	destPath := fileType + "s/" + bucket + "/" + filename

	// Set expiration duration, default to 15 minutes.
	expiry := 15 * time.Minute
	if expStr := c.Query("expires"); expStr != "" {
		if exp, err := time.ParseDuration(expStr); err == nil {
			expiry = exp
		}
	}

	url, err := h.StorageSvc.GetDownloadURL(c, destPath, expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download URL", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"downloadURL": url})
}
