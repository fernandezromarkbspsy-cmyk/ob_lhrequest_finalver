package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"golang-dashboard/internal/auth"
	"golang-dashboard/internal/database"
	"golang-dashboard/internal/models"

	"github.com/labstack/echo/v4"
)

// SeatalkLoginInitiate initiates the Seatalk OAuth flow and returns the authorization URL
func SeatalkLoginInitiate(c echo.Context) error {
	// Generate a random state for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate state")
	}

	// Store state in session (implement based on your session management)
	// For now, we'll return it in the response for client-side storage
	seatalkClient := auth.NewSeatalkClient()
	authURL := seatalkClient.GetAuthorizationURL(state)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"auth_url": authURL,
		"state":    state,
	})
}

// SeatalkLoginCallback handles the OAuth callback from Seatalk
func SeatalkLoginCallback(c echo.Context) error {
	code := c.QueryParam("code")
	state := c.QueryParam("state")

	if code == "" {
		return c.Render(http.StatusBadRequest, "login.html", map[string]string{
			"Error": "Authorization code not provided",
		})
	}

	// Verify state parameter (implement proper session-based verification)
	// This is a simplified example - in production, validate state against session
	_ = state

	seatalkClient := auth.NewSeatalkClient()

	// Exchange authorization code for access token
	tokenResp, err := seatalkClient.ExchangeCodeForToken(code)
	if err != nil {
		return c.Render(http.StatusBadRequest, "login.html", map[string]string{
			"Error": fmt.Sprintf("Failed to authenticate: %v", err),
		})
	}

	// Get user profile from Seatalk
	seatalkUser, err := seatalkClient.GetUserProfile(tokenResp.AccessToken)
	if err != nil {
		return c.Render(http.StatusBadRequest, "login.html", map[string]string{
			"Error": fmt.Sprintf("Failed to retrieve user profile: %v", err),
		})
	}

	// Find or create user in database
	var user models.User

	if database.DB != nil {
		result := database.DB.Where("email = ?", seatalkUser.Email).First(&user)

		if result.RowsAffected == 0 {
			// Create new user from Seatalk data
			// Default role is 'fte_ops', adjust based on business logic
			userEmail := seatalkUser.Email
			user = models.User{
				Name:     seatalkUser.Name,
				Email:    &userEmail,
				Role:     "fte_ops",
				IsFTE:    true,
				IsActive: true,
			}

			if err := database.DB.Create(&user).Error; err != nil {
				return c.Render(http.StatusInternalServerError, "login.html", map[string]string{
					"Error": "Failed to create user account",
				})
			}
		}
	}

	// Return user data (client will handle session storage)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"name":     user.Name,
		"email":    user.Email,
		"role":     user.Role,
		"is_fte":   user.IsFTE,
		"redirect": redirectForRole(user.Role),
	})
}

// SeatalkLoginAPI is an alternative API endpoint for one-tap login
// Client sends Seatalk token, server validates and creates/updates user
func SeatalkLoginAPI(c echo.Context) error {
	payload := struct {
		AccessToken string `json:"access_token"`
	}{}

	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid payload")
	}

	if payload.AccessToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Access token required")
	}

	seatalkClient := auth.NewSeatalkClient()

	// Get user profile from Seatalk using the provided token
	seatalkUser, err := seatalkClient.GetUserProfile(payload.AccessToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Invalid token: %v", err))
	}

	var user models.User

	if database.DB != nil {
		result := database.DB.Where("email = ?", seatalkUser.Email).First(&user)

		if result.RowsAffected == 0 {
			// Determine role based on phone number or other logic
			role := determineSeatalkUserRole(seatalkUser)

			userEmail := seatalkUser.Email
			user = models.User{
				Name:     seatalkUser.Name,
				Email:    &userEmail,
				Role:     role,
				IsFTE:    strings.HasPrefix(role, "fte_"),
				IsActive: true,
			}

			if err := database.DB.Create(&user).Error; err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create user")
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"name":     user.Name,
		"email":    user.Email,
		"role":     user.Role,
		"is_fte":   user.IsFTE,
		"redirect": redirectForRole(user.Role),
	})
}

// generateRandomState generates a random state string for CSRF protection
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// determineSeatalkUserRole determines user role based on Seatalk profile data
// This is a placeholder - customize based on your business logic
func determineSeatalkUserRole(user *auth.SeatalkUser) string {
	// Example: determine role based on phone number pattern or other criteria
	// For now, default to fte_ops
	// You could also query a mapping table or external service

	// If phone matches specific pattern, assign ops_pic role
	if user.Phone != "" && len(user.Phone) > 10 {
		// Custom logic here
		return "fte_ops"
	}

	return "fte_ops"
}
