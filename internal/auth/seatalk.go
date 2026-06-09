package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// SeatalkUser represents the user data from Seatalk API
type SeatalkUser struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Phone  string `json:"phone,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

// TokenResponse represents the OAuth token response from Seatalk
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// SeatalkClient handles Seatalk OAuth flow
type SeatalkClient struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	APIBase      string
}

// NewSeatalkClient creates a new Seatalk client instance
func NewSeatalkClient() *SeatalkClient {
	return &SeatalkClient{
		ClientID:     os.Getenv("SEATALK_CLIENT_ID"),
		ClientSecret: os.Getenv("SEATALK_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("SEATALK_REDIRECT_URI"),
		APIBase:      os.Getenv("SEATALK_API_BASE"),
	}
}

// GetAuthorizationURL returns the URL for QR code login
func (sc *SeatalkClient) GetAuthorizationURL(state string) string {
	params := url.Values{
		"client_id":     {sc.ClientID},
		"redirect_uri":  {sc.RedirectURI},
		"response_type": {"code"},
		"scope":         {"user.profile"},
		"state":         {state},
	}
	return fmt.Sprintf("%s/oauth/authorize?%s", sc.APIBase, params.Encode())
}

// ExchangeCodeForToken exchanges authorization code for access token
func (sc *SeatalkClient) ExchangeCodeForToken(code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {sc.ClientID},
		"client_secret": {sc.ClientSecret},
		"redirect_uri":  {sc.RedirectURI},
	}

	resp, err := http.PostForm(
		fmt.Sprintf("%s/oauth/token", sc.APIBase),
		data,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// GetUserProfile retrieves user profile from Seatalk API using access token
func (sc *SeatalkClient) GetUserProfile(accessToken string) (*SeatalkUser, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/api/user/profile", sc.APIBase),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get profile failed (status %d): %s", resp.StatusCode, string(body))
	}

	var user SeatalkUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user profile: %w", err)
	}

	return &user, nil
}

// RefreshAccessToken refreshes an expired access token
func (sc *SeatalkClient) RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {sc.ClientID},
		"client_secret": {sc.ClientSecret},
	}

	resp, err := http.PostForm(
		fmt.Sprintf("%s/oauth/token", sc.APIBase),
		data,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}
