package handlers

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"bloomify/config"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

const (
	MaxAudioFileSize = 20 * 1024 * 1024 // 20MB
	AllowedExtension = ".wav"
)

var allowedMIMETypes = []string{"audio/wav", "audio/x-wav"}

func validateAudioFile(header *multipart.FileHeader, mimeType string) error {
	if header.Size > MaxAudioFileSize {
		return errors.New("audio file size exceeds maximum allowed size")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != AllowedExtension {
		return errors.New("unsupported audio file format, only .wav files are allowed")
	}
	valid := false
	for _, mt := range allowedMIMETypes {
		if mimeType == mt {
			valid = true
			break
		}
	}
	if !valid {
		return errors.New("unsupported MIME type; please upload a valid WAV file")
	}
	return nil
}

func AISTTHandler(c *gin.Context) {
	// 1. Extract parameters.
	language := c.DefaultPostForm("language", "en-US")
	if loc, exists := c.Get("location"); exists {
		log.Printf("Location parameter: %v", loc)
	} else {
		log.Printf("No location parameter found in context.")
	}

	// 2. Retrieve the audio file.
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Audio file is required: " + err.Error()})
		return
	}
	defer file.Close()

	// 3. Validate file size, extension, and MIME type.
	// Attempt to obtain MIME type from header; if not provided, fallback to empty string.
	mimeType := header.Header.Get("Content-Type")
	if err := validateAudioFile(header, mimeType); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid audio file: " + err.Error()})
		return
	}

	// 4. Read audio file into memory.
	audioData, err := ioutil.ReadAll(io.LimitReader(file, MaxAudioFileSize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to read audio file: " + err.Error()})
		return
	}

	// 5. Initialize Google Speech-to-Text client.
	ctx := context.Background()
	client, err := speech.NewClient(ctx, option.WithCredentialsFile(config.AppConfig.GoogleServiceAccountFile))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create STT client: " + err.Error()})
		return
	}
	defer client.Close()

	// 6. Build the RecognizeRequest.
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    language,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioData,
			},
		},
	}

	// 7. Call the Speech-to-Text API.
	resp, err := client.Recognize(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "STT API error: " + err.Error()})
		return
	}

	// 8. Process the response.
	if len(resp.Results) == 0 {
		c.JSON(http.StatusOK, gin.H{"transcription": "", "message": "No speech detected."})
		return
	}

	var transcript string
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			transcript += alt.Transcript + " "
		}
	}

	// 9. Return only the transcription.
	c.JSON(http.StatusOK, gin.H{
		"transcription": transcript,
	})
}
