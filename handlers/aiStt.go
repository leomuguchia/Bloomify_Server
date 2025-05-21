package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"bloomify/config"

	speech "cloud.google.com/go/speech/apiv1"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

const (
	MaxDurationSeconds = 60              // 1 minute maximum
	MaxFileSize        = 5 * 1024 * 1024 // 5MB (conservative buffer)
)

var allowedExts = map[string]bool{
	".wav": true,
	".m4a": true,
	".mp3": true,
}

type waveHeader struct {
	RiffTag       [4]byte
	FileSize      uint32
	WaveTag       [4]byte
	FmtTag        [4]byte
	FmtSize       uint32
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	DataTag       [4]byte
	DataSize      uint32
}

// parseWaveHeader remains unchanged…

func convertAudio(inputPath, outputPath string) error {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in system PATH: %v", err)
	}

	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", inputPath,
		"-acodec", "pcm_s16le",
		"-ac", "1",
		"-ar", "16000",
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg conversion failed: %s", stderr.String())
	}
	return nil
}

func (h *DefaultAIHandler) AISTTHandler(c *gin.Context) {
	// 1. Language parameter (default en-US)
	language := c.DefaultPostForm("language", "en-US")

	// 2. Uploaded file
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio file is required", "details": err.Error()})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid file type",
			"details": fmt.Sprintf("allowed: .wav, .m4a, .mp3; got %s", ext),
		})
		return
	}

	// 3. Save original upload
	tempInput, err := os.CreateTemp("", "audio-*"+ext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp file", "details": err.Error()})
		return
	}
	defer os.Remove(tempInput.Name())
	defer tempInput.Close()

	if _, err := io.Copy(tempInput, io.LimitReader(file, MaxFileSize)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save audio file", "details": err.Error()})
		return
	}

	// 4. Prepare a WAV output file
	tempOutput, err := os.CreateTemp("", "converted-*.wav")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create output temp file", "details": err.Error()})
		return
	}
	defer os.Remove(tempOutput.Name())
	defer tempOutput.Close()

	// 5. If not already WAV, convert; otherwise just copy
	if ext != ".wav" {
		if err := convertAudio(tempInput.Name(), tempOutput.Name()); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "audio conversion failed", "details": err.Error()})
			return
		}
	} else {
		if _, err := tempInput.Seek(0, io.SeekStart); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "seek failed", "details": err.Error()})
			return
		}
		if _, err := io.Copy(tempOutput, tempInput); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to copy WAV", "details": err.Error()})
			return
		}
	}

	// 6. Read final WAV data
	audioData, err := os.ReadFile(tempOutput.Name())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read converted audio", "details": err.Error()})
		return
	}

	// 7. Speech client setup
	ctx := context.Background()
	client, err := speech.NewClient(ctx, option.WithCredentialsFile(config.AppConfig.GoogleServiceAccountFile))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to init speech client", "details": err.Error()})
		return
	}
	defer client.Close()

	// 8. Build and send RecognizeRequest
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:          speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz:   16000,
			LanguageCode:      language,
			AudioChannelCount: 1,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: audioData},
		},
	}

	resp, err := client.Recognize(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "speech recognition failed", "details": err.Error()})
		return
	}

	// 9. Aggregate transcript
	var transcript strings.Builder
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			transcript.WriteString(alt.Transcript + " ")
		}
	}

	c.JSON(http.StatusOK, gin.H{"transcription": strings.TrimSpace(transcript.String())})
}
