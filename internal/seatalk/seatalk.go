package seatalk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	APIBaseURL   string
}

type QRSessionResponse struct {
	SessionKey string `json:"session_key"`
	QRCodeURL  string `json:"qr_code_url"`
	ExpiresAt  int64  `json:"expires_at"`
}

type QRPollResponse struct {
	Status      string `json:"status"` // "pending", "authorized", "expired"
	AccessToken string `json:"access_token,omitempty"`
	UserID      string `json:"user_id,omitempty"`
}

type UserInfoResponse struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

var DefaultConfig Config

func init() {
	DefaultConfig = Config{
		ClientID:     os.Getenv("SEATALK_CLIENT_ID"),
		ClientSecret: os.Getenv("SEATALK_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("SEATALK_REDIRECT_URI"),
		APIBaseURL:   os.Getenv("SEATALK_API_BASE_URL"),
	}

	// Default values if not provided
	if DefaultConfig.APIBaseURL == "" {
		DefaultConfig.APIBaseURL = "https://api.seatalk.io"
	}
}

// CreateQRSession initiates a login session and returns a QR code
func CreateQRSession() (*QRSessionResponse, error) {
	if DefaultConfig.ClientID == "" || DefaultConfig.ClientSecret == "" {
		return nil, fmt.Errorf("seatalk configuration missing: client_id or client_secret")
	}

	url := DefaultConfig.APIBaseURL + "/openapi/auth/qr/session"
	
	payload := map[string]interface{}{
		"client_id":     DefaultConfig.ClientID,
		"client_secret": DefaultConfig.ClientSecret,
		"redirect_uri":  DefaultConfig.RedirectURI,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("seatalk API error: %d - %s", resp.StatusCode, string(data))
	}

	var result QRSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// PollQRStatus checks if user has authorized the QR code login
func PollQRStatus(sessionKey string) (*QRPollResponse, error) {
	url := fmt.Sprintf("%s/openapi/auth/qr/poll?session_key=%s&client_id=%s&client_secret=%s",
		DefaultConfig.APIBaseURL,
		sessionKey,
		DefaultConfig.ClientID,
		DefaultConfig.ClientSecret,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("seatalk API error: %d - %s", resp.StatusCode, string(data))
	}

	var result QRPollResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetUserInfo fetches user information using an access token
func GetUserInfo(accessToken string) (*UserInfoResponse, error) {
	url := DefaultConfig.APIBaseURL + "/openapi/user/info"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("seatalk API error: %d - %s", resp.StatusCode, string(data))
	}

	var result UserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ExchangeCodeForToken exchanges an authorization code for an access token
func ExchangeCodeForToken(code string) (string, error) {
	url := DefaultConfig.APIBaseURL + "/openapi/auth/token"

	payload := map[string]interface{}{
		"client_id":     DefaultConfig.ClientID,
		"client_secret": DefaultConfig.ClientSecret,
		"code":          code,
		"grant_type":    "authorization_code",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("seatalk API error: %d - %s", resp.StatusCode, string(data))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("invalid access_token in response")
	}

	return accessToken, nil
}
