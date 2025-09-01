package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ===== Config Structs =====
type AllowedPath struct {
	Path    string   `json:"path" yaml:"path"`
	Methods []string `json:"methods" yaml:"methods"` // e.g. ["GET","POST"]
}

type Endpoint struct {
	Name        string        `json:"name" yaml:"name"`
	TargetURL   string        `json:"target_url" yaml:"target_url"`
	AuthType    string        `json:"auth_type" yaml:"auth_type"` // "bearer" or "none"
	AuthAPI     string        `json:"auth_api" yaml:"auth_api"`
	Username    string        `json:"username" yaml:"username"`
	PasswordEnc string        `json:"password_enc" yaml:"password_enc"`
	Allowed     []AllowedPath `json:"allowed" yaml:"allowed"` // whitelist of allowed paths & methods
}

type Config struct {
	Endpoints []Endpoint `json:"endpoints" yaml:"endpoints"`
}

// ===== Load Config =====
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if strings.HasSuffix(path, ".json") {
		err = json.Unmarshal(data, &cfg)
	} else {
		err = yaml.Unmarshal(data, &cfg)
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
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

	// Expect { "access_token": "...", "expires_in": 3600 }
	var parsed map[string]any
	if err := json.Unmarshal(respData, &parsed); err != nil {
		return "", err
	}

	token, ok := parsed["access_token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("missing access_token in response for %s", e.Name)
	}

	// default 1h
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

// ===== Proxy Transport =====
type authTransport struct {
	base     http.RoundTripper
	endpoint Endpoint
	authMgr  *AuthManager
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.endpoint.AuthType == "bearer" {
		token, err := t.authMgr.GetToken(t.endpoint)
		if err != nil {
			log.Printf("Auth error for %s: %v", t.endpoint.Name, err)
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return t.base.RoundTrip(req)
}

// ===== Reverse Proxy Builder =====
func newProxy(e Endpoint, authMgr *AuthManager) *httputil.ReverseProxy {
	target, err := url.Parse(e.TargetURL)
	if err != nil {
		log.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &authTransport{base: http.DefaultTransport, endpoint: e, authMgr: authMgr}

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api/"+e.Name)
	}
	return proxy
}

// ===== Restricted Proxy Handler =====
func newRestrictedProxy(e Endpoint, authMgr *AuthManager) http.Handler {
	proxy := newProxy(e, authMgr)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/"+e.Name)

		allowed := false
		for _, a := range e.Allowed {
			if strings.HasPrefix(path, a.Path) {
				for _, m := range a.Methods {
					if strings.EqualFold(r.Method, m) {
						allowed = true
						break
					}
				}
				if allowed {
					break
				}
			}
		}

		if !allowed {
			http.Error(w, "Forbidden: endpoint or method not allowed", http.StatusForbidden)
			return
		}

		proxy.ServeHTTP(w, r)
	})
}

// ===== Main =====
func main() {
	configFile := flag.String("config", "config.yaml", "Path to config file (YAML or JSON)")
	port := flag.String("port", "8080", "Port to listen on")
	flag.Parse()

	secretKey := os.Getenv("MASTER_KEY")
	if len(secretKey) != 32 {
		log.Fatal("MASTER_KEY must be set to 32 characters (AES-256 key)")
	}

	cfg, err := LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	authMgr := NewAuthManager(secretKey)

	for _, e := range cfg.Endpoints {
		prefix := "/api/" + e.Name
		http.Handle(prefix+"/", newRestrictedProxy(e, authMgr))
		log.Printf("Mapped %s -> %s (allowed paths=%v)", prefix, e.TargetURL, e.Allowed)
	}

	log.Printf("Proxy server listening on :%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
