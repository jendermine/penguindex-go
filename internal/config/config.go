// File: penguindex-go/internal/config/config.go
package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256" // Or crypto/sha512 if preferred for PBKDF2
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/crypto/pbkdf2"
)

const DEFAULT_TEST_FOLDER_ID = "1y1OzjZ5zrzX8rCNza6evRz68UHlSYuIW"

// Crypto parameters for PBKDF2 and AES-GCM
const (
	PBKDF2_ITERATIONS = 310000

	DERIVED_KEY_SIZE   uint32 = 32 // AES-256 key size
	SALT_SIZE          = 16
	NONCE_SIZE_AES_GCM = 12 // Standard AES-GCM nonce size
)

type RemoteEncryptedBundle struct {
	EncryptedBundle string `json:"encrypted_bundle"` // hex encoded "salt + nonce + ciphertext"
}

type DecryptedBundle struct {
	ServiceAccountJSONString string `json:"service_account_json_string"`
	TelegramBotToken         string `json:"telegram_bot_token"`
}

type AppConfig struct {
	ServiceAccountJSON string
	TelegramBotToken   string
	TelegramChatID     string
	DefaultFolderID    string
}

type RemoteConfigDetails struct {
	EncryptedBundleHex string
	TelegramChatID     string
}

func FetchRemoteConfigDetails(bundleURL, telegramChatIDURL string) (*RemoteConfigDetails, error) {
	details := &RemoteConfigDetails{}

	respBundle, err := http.Get(bundleURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get bundle URL %s: %w", bundleURL, err)
	}
	defer respBundle.Body.Close()
	if respBundle.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status from bundle URL %s: %s", bundleURL, respBundle.Status)
	}
	bodyBundleBytes, err := io.ReadAll(respBundle.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from bundle URL %s: %w", bundleURL, err)
	}
	var remoteBundle RemoteEncryptedBundle
	if err := json.Unmarshal(bodyBundleBytes, &remoteBundle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encrypted bundle JSON from %s: %w", bundleURL, err)
	}
	details.EncryptedBundleHex = remoteBundle.EncryptedBundle

	respChatID, err := http.Get(telegramChatIDURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get Telegram chat ID URL %s: %w", telegramChatIDURL, err)
	}
	defer respChatID.Body.Close()
	if respChatID.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status from Telegram chat ID URL %s: %s", telegramChatIDURL, respChatID.Status)
	}
	bodyChatIDBytes, err := io.ReadAll(respChatID.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from Telegram chat ID URL %s: %w", telegramChatIDURL, err)
	}
	details.TelegramChatID = string(bodyChatIDBytes)

	return details, nil
}

func DecryptBundle(hexEncodedEncryptedBundle, pin string) (*DecryptedBundle, error) {
	encryptedBundleBytes, err := hex.DecodeString(hexEncodedEncryptedBundle)
	if err != nil {
		return nil, fmt.Errorf("failed to hex decode encrypted bundle: %w", err)
	}

	if len(encryptedBundleBytes) < SALT_SIZE+NONCE_SIZE_AES_GCM {
		return nil, fmt.Errorf("encrypted data too short, expected salt + nonce + ciphertext, got %d bytes", len(encryptedBundleBytes))
	}

	salt := encryptedBundleBytes[:SALT_SIZE]
	nonce := encryptedBundleBytes[SALT_SIZE : SALT_SIZE+NONCE_SIZE_AES_GCM]
	ciphertext := encryptedBundleBytes[SALT_SIZE+NONCE_SIZE_AES_GCM:]

	derivedKey := pbkdf2.Key([]byte(pin), salt, PBKDF2_ITERATIONS, int(DERIVED_KEY_SIZE), sha256.New)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(nonce) != aesgcm.NonceSize() {
		return nil, fmt.Errorf("incorrect nonce size: expected %d, got %d", aesgcm.NonceSize(), len(nonce))
	}

	decryptedData, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// This error is common for incorrect PINs or corrupted data.
		return nil, fmt.Errorf("failed to decrypt bundle (check PIN or bundle content - PBKDF2 used): %w", err)
	}

	var bundle DecryptedBundle
	if err := json.Unmarshal(decryptedData, &bundle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decrypted bundle JSON: %w", err)
	}

	return &bundle, nil
}
