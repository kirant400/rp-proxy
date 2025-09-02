package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ===== Auth Manager =====
type AuthToken struct {
	Token     string
	ExpiresAt time.Time
}

type AuthManager struct {
	mu     sync.Mutex
	tokens map[string]*AuthToken
	key    string
}

func NewAuthManager(secretKey string) *AuthManager {
	return &AuthManager{tokens: make(map[string]*AuthToken), key: secretKey}
}

func (am *AuthManager) GetToken(e Endpoint) (string, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// If token exists and not expired, reuse it
	if tok, ok := am.tokens[e.Name]; ok && time.Now().Before(tok.ExpiresAt) {
		return tok.Token, nil
	}

	// Decrypt password
	plainPassword, err := DecryptAESGCM(am.key, e.PasswordEnc)
	if err != nil {
		return "", fmt.Errorf("decrypt password for %s: %w", e.Name, err)
	}

	// Call auth API
	body := map[string]string{
		"username": e.Username,
		"password": plainPassword,
	}
	jsonData, _ := json.Marshal(body)

	resp, err := http.Post(e.AuthAPI, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		respData, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth API failed for %s: %s", e.Name, string(respData))
	}

	respData, _ := io.ReadAll(resp.Body)

	// Expect { "token": "...", "expires_in": 3600 }
	var parsed map[string]any
	if err := json.Unmarshal(respData, &parsed); err != nil {
		return "", err
	}

	token, ok := parsed["token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("missing access_token in response for %s", e.Name)
	}

	// default 1h if not provided
	expiresIn := int64(3600)
	if v, ok := parsed["expires_in"].(float64); ok {
		expiresIn = int64(v)
	}

	am.tokens[e.Name] = &AuthToken{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Duration(expiresIn-60) * time.Second), // refresh 1 min early
	}

	return token, nil
}

// ===== AES-GCM Decrypt =====
func DecryptAESGCM(secretKey, cipherB64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}
	block, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce := raw[:gcm.NonceSize()]
	ct := raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("gcm open: %w", err)
	}
	return string(pt), nil
}
