package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

// Helper: Encrypt with AES-GCM for tests
func EncryptAESGCM(secretKey, plaintext string) (string, error) {
	block, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Helper: Check path/method restriction
func AllowedRequest(e Endpoint, path, method string) bool {
	for _, a := range e.Allowed {
		if len(path) >= len(a.Path) && path[:len(a.Path)] == a.Path {
			for _, m := range a.Methods {
				if m == method {
					return true
				}
			}
		}
	}
	return false
}
