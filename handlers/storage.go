package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bloomify/services/storage"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

const internalAdminKey = "mUGuchIa_LIO"

type uploadRequest struct {
	FileType string `json:"fileType"` // For general uploads: "image" or "video"
	Bucket   string `json:"bucket"`   // e.g. "images", "videos", "documents", "selfies"
	Filename string `json:"filename"` // For download URL requests
}

type StorageHandler struct {
	StorageSvc storage.StorageService
	AdminKey   string
}

func NewStorageHandler(svc storage.StorageService) *StorageHandler {
	adminKey := viper.GetString("cloudinary.adminKey")
	return &StorageHandler{
		StorageSvc: svc,
		AdminKey:   adminKey,
	}
}

var allowedBuckets = map[string]bool{
	"images":  true,
	"videos":  true,
	"profile": true,
}

var allowedKYPBuckets = map[string]bool{
	"documents": true,
	"selfies":   true,
}

func (h *StorageHandler) UploadFileHandler(c *gin.Context) {
	fileType := c.PostForm("fileType")
	bucket := c.PostForm("bucket")

	log.Printf("[UploadFileHandler] Received request - fileType: %s, bucket: %s", fileType, bucket)

	if fileType != "image" && fileType != "video" {
		log.Printf("[UploadFileHandler] Invalid fileType: %s", fileType)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type; must be 'image' or 'video'"})
		return
	}

	if !allowedBuckets[bucket] {
		log.Printf("[UploadFileHandler] Invalid bucket: %s", bucket)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket; allowed values are 'images', 'videos', 'profile'"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		log.Printf("[UploadFileHandler] Failed to retrieve file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not provided", "detail": err.Error()})
		return
	}

	log.Printf("[UploadFileHandler] File received - name: %s, size: %d", fileHeader.Filename, fileHeader.Size)

	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, fileHeader.Filename)
	log.Printf("[UploadFileHandler] Saving file to temporary path: %s", tempFilePath)

	if err := c.SaveUploadedFile(fileHeader, tempFilePath); err != nil {
		log.Printf("[UploadFileHandler] Failed to save uploaded file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file", "detail": err.Error()})
		return
	}
	defer func() {
		log.Printf("[UploadFileHandler] Cleaning up temporary file: %s", tempFilePath)
		os.Remove(tempFilePath)
	}()

	destFolder := "public/" + bucket
	log.Printf("[UploadFileHandler] Uploading file to storage - folder: %s", destFolder)

	publicID, err := h.StorageSvc.UploadFile(c.Request.Context(), tempFilePath, destFolder)
	if err != nil {
		log.Printf("[UploadFileHandler] Failed to upload file to storage: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file", "detail": err.Error()})
		return
	}

	log.Printf("[UploadFileHandler] File uploaded successfully - publicID: %s", publicID)

	downloadURL, err := h.StorageSvc.GetDownloadURL(c.Request.Context(), publicID, 0)
	if err != nil {
		log.Printf("[UploadFileHandler] Failed to get download URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to construct download URL", "detail": err.Error()})
		return
	}

	log.Printf("[UploadFileHandler] Download URL generated: %s", downloadURL)

	c.JSON(http.StatusOK, gin.H{
		"message":     "file uploaded successfully",
		"downloadURL": downloadURL,
		"publicID":    publicID,
	})
}

func (h *StorageHandler) GetDownloadURLHandler(c *gin.Context) {
	var req uploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "detail": err.Error()})
		return
	}

	if !allowedBuckets[req.Bucket] || req.Filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket or filename"})
		return
	}

	destPath := "public/" + req.Bucket + "/" + req.Filename

	expiry := 15 * time.Minute
	if expStr := c.Query("expires"); expStr != "" {
		if exp, err := time.ParseDuration(expStr); err == nil {
			expiry = exp
		}
	}

	url, err := h.StorageSvc.GetDownloadURL(c.Request.Context(), destPath, expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download URL", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"downloadURL": url})
}

func (h *StorageHandler) KYPUploadFileHandler(c *gin.Context) {
	bucket := c.PostForm("bucket")
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

	isImage := bucket == "selfies"

	publicID, err := h.StorageSvc.(*storage.FirebaseStorageService).UploadWithContext(
		c.Request.Context(), tempFilePath, true, true, isImage, internalAdminKey,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload KYP file", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "KYP file uploaded successfully",
		"publicID": publicID,
	})
}

func (h *StorageHandler) KYPGetDownloadURLHandler(c *gin.Context) {
	adminToken, exists := c.Get("adminToken")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "admin token not found"})
		return
	}

	if tokenStr, ok := adminToken.(string); !ok || tokenStr != internalAdminKey {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin key"})
		return
	}

	var req uploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body", "detail": err.Error()})
		return
	}

	if !allowedKYPBuckets[req.Bucket] || req.Filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bucket or filename"})
		return
	}

	destPath := "private/kyp/"
	if req.Bucket == "selfies" {
		destPath += "images/"
	} else {
		destPath += "files/"
	}
	destPath += req.Filename

	expiry := 15 * time.Minute
	if expStr := c.Query("expires"); expStr != "" {
		if exp, err := time.ParseDuration(expStr); err == nil {
			expiry = exp
		}
	}

	secureURL, err := h.StorageSvc.GetSecureDownloadURL(c.Request.Context(), destPath, expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate secure download URL", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"downloadURL": secureURL})
}
