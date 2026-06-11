package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang-dashboard/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) Login(payload LoginPayload) (models.User, map[string]interface{}, error) {
	loginType := strings.ToLower(strings.TrimSpace(payload.LoginType))
	email := strings.TrimSpace(payload.Email)
	opsID := strings.TrimSpace(payload.OpsID)
	if !s.repo.Available() {
		user := demoUser(loginType, email, opsID)
		return user, loginResponse(user), nil
	}

	switch loginType {
	case "fte":
		if email == "" {
			return models.User{}, nil, AppError{Code: http.StatusBadRequest, Message: "Email is required"}
		}
	case "backroom":
		if opsID == "" {
			return models.User{}, nil, AppError{Code: http.StatusBadRequest, Message: "Ops ID is required"}
		}
	default:
		return models.User{}, nil, AppError{Code: http.StatusBadRequest, Message: "Choose FTE or Backroom"}
	}

	user, err := s.repo.FindLoginUser(loginType, email, opsID)
	if err != nil {
		return models.User{}, nil, AppError{Code: http.StatusUnauthorized, Message: "Invalid credentials"}
	}
	if user.PasswordHash != nil && *user.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(payload.Password)); err != nil {
			return models.User{}, nil, AppError{Code: http.StatusUnauthorized, Message: "Invalid credentials"}
		}
	}
	s.ensureUserUniqueID(&user)
	return user, loginResponse(user), nil
}

func (s Service) SendOTP(payload OTPPayload) (map[string]interface{}, error) {
	if !s.repo.Available() {
		return nil, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	email := strings.TrimSpace(payload.Email)
	if email == "" {
		return nil, AppError{Code: http.StatusBadRequest, Message: "Email is required"}
	}
	count, err := s.repo.CountActiveFTEByEmail(email)
	if err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to validate email"}
	}
	if count == 0 {
		return nil, AppError{Code: http.StatusUnauthorized, Message: "Invalid email"}
	}

	code, err := randomOTP()
	if err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to generate OTP"}
	}
	codeHash, err := hashPassword(code)
	if err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to store OTP"}
	}
	if err := s.repo.CreateOTP(&models.UserOTP{
		Email:     email,
		CodeHash:  codeHash,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}); err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to store OTP"}
	}
	if os.Getenv("APP_ENV") != "production" {
		fmt.Printf("OTP for %s: %s\n", email, code)
	}
	return map[string]interface{}{"ok": true, "expires_in_seconds": 600}, nil
}

func (s Service) VerifyOTP(payload OTPPayload) (models.User, map[string]interface{}, error) {
	if !s.repo.Available() {
		return models.User{}, nil, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	email := strings.TrimSpace(payload.Email)
	code := strings.TrimSpace(payload.Code)
	if email == "" || code == "" {
		return models.User{}, nil, AppError{Code: http.StatusBadRequest, Message: "Email and OTP are required"}
	}
	otp, err := s.repo.LatestValidOTP(email, time.Now())
	if err != nil {
		return models.User{}, nil, AppError{Code: http.StatusUnauthorized, Message: "Invalid OTP"}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(otp.CodeHash), []byte(code)); err != nil {
		return models.User{}, nil, AppError{Code: http.StatusUnauthorized, Message: "Invalid OTP"}
	}
	user, err := s.repo.ActiveFTEByEmail(email)
	if err != nil {
		return models.User{}, nil, AppError{Code: http.StatusUnauthorized, Message: "Invalid email"}
	}
	now := time.Now()
	_ = s.repo.MarkOTPUsed(&otp, now)
	s.ensureUserUniqueID(&user)
	return user, loginResponse(user), nil
}

func (s Service) ChangePassword(userID uint, payload ChangePasswordPayload) error {
	if !s.repo.Available() {
		return AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	if strings.TrimSpace(payload.NewPassword) == "" {
		return AppError{Code: http.StatusBadRequest, Message: "New password is required"}
	}
	user, err := s.repo.FindUser(userID)
	if err != nil {
		return AppError{Code: http.StatusNotFound, Message: "User not found"}
	}
	if user.PasswordHash != nil && *user.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(payload.CurrentPassword)); err != nil {
			return AppError{Code: http.StatusUnauthorized, Message: "Invalid current password"}
		}
	}
	passwordHash, err := hashPassword(payload.NewPassword)
	if err != nil {
		return AppError{Code: http.StatusBadRequest, Message: "Password is invalid"}
	}
	user.PasswordHash = stringPtr(passwordHash)
	user.FirstTimeLogin = false
	if err := s.repo.SaveUser(&user); err != nil {
		return AppError{Code: http.StatusInternalServerError, Message: "Unable to change password"}
	}
	return nil
}

func (s Service) ensureUserUniqueID(user *models.User) {
	if strings.TrimSpace(user.UniqueID) != "" {
		return
	}
	prefix := "BR"
	if isFTERole(user.Role) {
		prefix = "FTE"
	}
	user.UniqueID = prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if s.repo.Available() && user.ID > 0 {
		_ = s.repo.EnsureUniqueID(user)
	}
}

func SignSessionToken(claims map[string]interface{}) (string, error) {
	headerJSON, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := header + "." + payload
	return unsigned + "." + signJWTPart(unsigned), nil
}

func ReadSessionClaims(token string) (map[string]interface{}, bool) {
	if strings.TrimSpace(token) == "" {
		return nil, false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, false
	}
	unsigned := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(signJWTPart(unsigned))) {
		return nil, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	claims := map[string]interface{}{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, false
	}
	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return nil, false
	}
	return claims, true
}

func loginResponse(user models.User) map[string]interface{} {
	return map[string]interface{}{
		"id":        user.ID,
		"unique_id": user.UniqueID,
		"name":      user.Name,
		"role":      user.Role,
		"email":     user.Email,
		"ops_id":    user.OpsID,
		"is_fte":    isFTERole(user.Role),
		"redirect":  redirectForRole(user.Role),
	}
}

func demoUser(loginType, email, opsID string) models.User {
	user := models.User{Role: "ops_pic", Name: "Backroom Demo", Email: stringPtr(email), OpsID: stringPtr(opsID)}
	if loginType == "fte" {
		user.Role = "fte_ops"
		user.Name = "FTE Ops Demo"
		if strings.Contains(strings.ToLower(email), "mm") {
			user.Role = "fte_mm"
			user.Name = "FTE MM Demo"
		}
	}
	if loginType == "backroom" && (strings.Contains(strings.ToLower(opsID), "dock") || strings.Contains(strings.ToLower(opsID), "doc")) {
		user.Role = "dock_officer"
		user.Name = "Dock Officer Demo"
	}
	return user
}

func redirectForRole(role string) string {
	switch role {
	case "fte_mm":
		return "/midmile/truck-request"
	case "dock_officer", "doc_officer":
		return "/dock/officer"
	default:
		return "/dashboard"
	}
}

func isFTERole(role string) bool {
	return role == "fte_ops" || role == "fte_mm"
}

func numericClaim(value interface{}) (uint, bool) {
	switch typed := value.(type) {
	case float64:
		return uint(typed), typed > 0
	case int:
		return uint(typed), typed > 0
	case uint:
		return typed, typed > 0
	default:
		return 0, false
	}
}

func hashPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return "", nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func randomOTP() (string, error) {
	max := big.NewInt(1000000)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func signJWTPart(unsigned string) string {
	mac := hmac.New(sha256.New, []byte(sessionSecret()))
	_, _ = mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func sessionSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	if secret := os.Getenv("APP_SECRET"); secret != "" {
		return secret
	}
	return "soc5-dev-session-secret"
}

func stringPtr(value string) *string {
	return &value
}
