package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// encryptFile encrypts the file at localFilePath using AES-256 GCM with the given adminKey.
// It writes the encrypted data to a temporary file and returns the temporary file's path.
// The returned file contains the nonce prepended to the ciphertext.
func encryptFile(localFilePath, adminKey string) (string, error) {
	// Read the plaintext file.
	plaintext, err := ioutil.ReadFile(localFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Derive a 32-byte key from the adminKey using SHA-256.
	keyHash := sha256.Sum256([]byte(adminKey))
	key := keyHash[:] // 32 bytes for AES-256

	// Create the AES cipher.
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode instance.
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a nonce of the required size.
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext using GCM.
	// The nonce is prepended to the ciphertext so it can be used during decryption.
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Create a temporary file to store the encrypted data.
	tempDir := os.TempDir()
	// Use a unique filename within the tempDir.
	tempFilePath := filepath.Join(tempDir, fmt.Sprintf("enc-%d", time.Now().UnixNano()))
	if err := ioutil.WriteFile(tempFilePath, ciphertext, 0644); err != nil {
		return "", fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return tempFilePath, nil
}
