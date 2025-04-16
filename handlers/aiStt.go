package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
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
	AllowedExtension   = ".wav"
)

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

func parseWaveHeader(data []byte) (*waveHeader, error) {
	if len(data) < 44 {
		return nil, errors.New("invalid WAV header length")
	}

	var header waveHeader
	buf := bytes.NewReader(data)

	if err := binary.Read(buf, binary.LittleEndian, &header.RiffTag); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.FileSize); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.WaveTag); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.FmtTag); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.FmtSize); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.AudioFormat); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.NumChannels); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.SampleRate); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.ByteRate); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.BlockAlign); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.BitsPerSample); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.DataTag); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.DataSize); err != nil {
		return nil, err
	}

	return &header, nil
}

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

func AISTTHandler(c *gin.Context) {
	// 1. Get language parameter (default to en-US)
	language := c.DefaultPostForm("language", "en-US")

	// 2. Get audio file from multipart form
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "audio file is required",
			"details": err.Error(),
		})
		return
	}
	defer file.Close()

	// 3. Validate file extension
	if ext := strings.ToLower(filepath.Ext(header.Filename)); ext != AllowedExtension {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid file type",
			"details": fmt.Sprintf("expected %s, got %s", AllowedExtension, ext),
		})
		return
	}

	// 4. Create temp file for original audio
	tempInput, err := os.CreateTemp("", "audio-*.wav")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to create temp file",
			"details": err.Error(),
		})
		return
	}
	defer os.Remove(tempInput.Name())
	defer tempInput.Close()

	// 5. Save uploaded file to temp location
	if _, err := io.Copy(tempInput, io.LimitReader(file, MaxFileSize)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to save audio file",
			"details": err.Error(),
		})
		return
	}

	// 6. Create temp file for converted audio
	tempOutput, err := os.CreateTemp("", "converted-*.wav")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to create output temp file",
			"details": err.Error(),
		})
		return
	}
	defer os.Remove(tempOutput.Name())
	defer tempOutput.Close()

	// 7. Convert audio to proper format
	if err := convertAudio(tempInput.Name(), tempOutput.Name()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "audio conversion failed",
			"details": err.Error(),
		})
		return
	}

	// 8. Read converted audio data
	audioData, err := os.ReadFile(tempOutput.Name())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to read converted audio",
			"details": err.Error(),
		})
		return
	}

	// 9. Initialize Google STT client
	ctx := context.Background()
	client, err := speech.NewClient(ctx, option.WithCredentialsFile(config.AppConfig.GoogleServiceAccountFile))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to initialize speech client",
			"details": err.Error(),
		})
		return
	}
	defer client.Close()

	// 10. Configure recognition request
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:          speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz:   16000,
			LanguageCode:      language,
			AudioChannelCount: 1, // Mono
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioData,
			},
		},
	}

	// 11. Process with Google STT
	resp, err := client.Recognize(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "speech recognition failed",
			"details": err.Error(),
		})
		return
	}

	// 12. Format results
	var transcript strings.Builder
	if len(resp.Results) > 0 {
		for _, result := range resp.Results {
			for _, alt := range result.Alternatives {
				transcript.WriteString(alt.Transcript + " ")
			}
		}
	}

	// 13. Return successful response
	c.JSON(http.StatusOK, gin.H{
		"transcription": strings.TrimSpace(transcript.String()),
	})
}
