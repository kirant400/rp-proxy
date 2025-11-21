package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Sample AES-256 key (32 chars)
var testKey = "12345678901234567890123456789012"

// helper to write temp file
func writeTempFile(t *testing.T, content string, ext string) string {
	t.Helper()
	tmp := filepath.Join(os.TempDir(), "proxytest."+ext)
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return tmp
}

func TestLoadConfigYAML(t *testing.T) {
	content := `
endpoints:
  - name: service1
    target_url: https://api.test.com
    auth_type: bearer
    auth_api: https://auth.test.com/token
    username: testuser
    password_enc: "dummy"
    allowed:
      - path: /users
        methods: ["GET","POST"]
`
	file := writeTempFile(t, content, "yaml")
	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if len(cfg.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(cfg.Endpoints))
	}
	if cfg.Endpoints[0].Allowed[0].Path != "/users" {
		t.Errorf("expected path /users, got %s", cfg.Endpoints[0].Allowed[0].Path)
	}
}

func TestLoadConfigJSON(t *testing.T) {
	content := `{
		"endpoints": [{
			"name": "public",
			"target_url": "https://api.public.com",
			"auth_type": "none",
			"allowed": [
				{"path": "/ping", "methods": ["GET"]}
			]
		}]
	}`
	file := writeTempFile(t, content, "json")
	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Endpoints[0].Name != "public" {
		t.Errorf("expected endpoint 'public', got %s", cfg.Endpoints[0].Name)
	}
}

// Validate path + method restriction
func TestPathRestriction(t *testing.T) {
	e := Endpoint{
		Name: "svc",
		Allowed: []AllowedPath{
			{Path: "/users", Methods: []string{"GET"}},
			{Path: "/orders", Methods: []string{"POST"}},
		},
	}

	// Allowed: GET /users
	req1 := AllowedRequest(e, "/users/list", "GET")
	if !req1 {
		t.Error("expected GET /users to be allowed")
	}

	// Forbidden: POST /users
	req2 := AllowedRequest(e, "/users/list", "POST")
	if req2 {
		t.Error("expected POST /users to be forbidden")
	}

	// Allowed: POST /orders
	req3 := AllowedRequest(e, "/orders", "POST")
	if !req3 {
		t.Error("expected POST /orders to be allowed")
	}
}

// Validate AES-GCM roundtrip
func TestDecryptAESGCM(t *testing.T) {
	// Encrypt "mypassword" with testKey
	cipher, err := EncryptAESGCM(testKey, "mypassword")
	if err != nil {
		t.Fatalf("EncryptAESGCM failed: %v", err)
	}

	plain, err := DecryptAESGCM(testKey, cipher)
	if err != nil {
		t.Fatalf("DecryptAESGCM failed: %v", err)
	}
	if plain != "mypassword" {
		t.Errorf("expected 'mypassword', got %s", plain)
	}
}
