// callback/sender.go
package callback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kirant400/rp-wrapper/config"
)

var (
	cachedToken      string
	cachedAuthType   string
	tokenExpiresAt   int64
)

type AttendancePayload struct {
	StationID  string `json:"station_id"`
	AreaID     string `json:"area_id,omitempty"`
	Timestamp  int64  `json:"timestamp"`
	Event      string `json:"event"`
	SourceIP   string `json:"source_ip,omitempty"`
	WrapperURL string `json:"wrapper_url,omitempty"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
}

// getAccessToken fetches a fresh token using any of the supported auth methods
func getAccessToken() (string, error) {
	te := config.Global.AttendanceCallback.TokenEndpoint
	if te == nil {
		return "", nil // no token endpoint configured
	}

	// Reuse cached token if still valid (30s safety buffer)
	if cachedToken != "" && time.Now().Unix() < tokenExpiresAt-30 {
		return cachedToken, nil
	}

	method := http.MethodPost
	if te.Method != "" {
		method = te.Method
	}

	// Build form body
	data := url.Values{}
	for k, v := range te.ExtraParams {
		data.Set(k, v)
	}
	if te.Username != "" {
		data.Set("username", te.Username)
	}
	if te.Password != "" {
		data.Set("password", te.Password)
	}
	// ── Authentication priority ──
	if te.AuthorizationKey != "" {
		data.Set("authorizationKey", te.AuthorizationKey)
	} 

	req, _ := http.NewRequest(method, te.URL, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/json")

	
	// 3. Form username/password → already added above

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}

	if tr.AccessToken == "" {
		return "", fmt.Errorf("no access_token in response")
	}

	// Cache the token
	cachedToken = tr.AccessToken
	expiresIn := 3600
	if tr.ExpiresIn > 0 {
		expiresIn = tr.ExpiresIn
	}
	tokenExpiresAt = time.Now().Unix() + int64(expiresIn)

	return cachedToken, nil
}

// SendAttendanceCallback is called every time someone clocks in
func SendAttendanceCallback(details map[string]interface{}) {
	if !config.Global.AttendanceCallback.Enabled || config.Global.AttendanceCallback.URL == "" {
		return
	}

	jsonData, _ := json.Marshal(details)

	timeout := 5
	if config.Global.AttendanceCallback.TimeoutSeconds > 0 {
		timeout = config.Global.AttendanceCallback.TimeoutSeconds
	}
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	retry := 1
	if config.Global.AttendanceCallback.RetryCount > 0 {
		retry += config.Global.AttendanceCallback.RetryCount
	}

	for attempt := 0; attempt < retry; attempt++ {
		// Always get fresh token (especially on 401)
		token, err := getAccessToken()

		req, _ := http.NewRequest("POST", config.Global.AttendanceCallback.URL, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		if err == nil && token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			if resp.StatusCode < 400 || (resp.StatusCode != 401 && resp.StatusCode != 403) {
				resp.Body.Close()
				return // success
			}
		}
		if resp != nil {
			resp.Body.Close()
		}

		// Invalidate token on auth failure
		if resp != nil && (resp.StatusCode == 401 || resp.StatusCode == 403) {
			cachedToken = ""
			tokenExpiresAt = 0
		}

		if attempt < retry-1 {
			time.Sleep(500 * time.Millisecond << attempt) // exponential backoff
		}
	}
	// Failed after retries → fire-and-forget (logged elsewhere if you add logging)
}