package auth

import (
	"net/http"
	"os"
	"time"

	"golang-dashboard/internal/database"
	"golang-dashboard/internal/models"

	"github.com/labstack/echo/v4"
)

type Controller struct {
	service Service
}

func NewController() Controller {
	return Controller{service: NewService(NewRepository(database.DB))}
}

func (c Controller) LoginAPI(ctx echo.Context) error {
	payload := LoginPayload{}
	if err := ctx.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid login payload")
	}
	user, response, err := c.service.Login(payload)
	if err != nil {
		return toHTTPError(err)
	}
	setSessionCookie(ctx, user)
	return ctx.JSON(http.StatusOK, response)
}

func (c Controller) LogoutAPI(ctx echo.Context) error {
	ctx.SetCookie(&http.Cookie{
		Name:     "soc5_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("APP_ENV") == "production",
	})
	return ctx.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func (c Controller) MeAPI(ctx echo.Context) error {
	claims, ok := readSessionClaims(ctx)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not signed in")
	}
	return ctx.JSON(http.StatusOK, claims)
}

func (c Controller) SendOTPAPI(ctx echo.Context) error {
	payload := OTPPayload{}
	if err := ctx.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid OTP payload")
	}
	response, err := c.service.SendOTP(payload)
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, response)
}

func (c Controller) VerifyOTPAPI(ctx echo.Context) error {
	payload := OTPPayload{}
	if err := ctx.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid OTP payload")
	}
	user, response, err := c.service.VerifyOTP(payload)
	if err != nil {
		return toHTTPError(err)
	}
	setSessionCookie(ctx, user)
	return ctx.JSON(http.StatusOK, response)
}

func (c Controller) ChangePasswordAPI(ctx echo.Context) error {
	claims, ok := readSessionClaims(ctx)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not signed in")
	}
	userID, ok := numericClaim(claims["id"])
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid session")
	}
	payload := ChangePasswordPayload{}
	if err := ctx.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid password payload")
	}
	if err := c.service.ChangePassword(userID, payload); err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func setSessionCookie(ctx echo.Context, user models.User) {
	token, err := SignSessionToken(map[string]interface{}{
		"id":        user.ID,
		"unique_id": user.UniqueID,
		"name":      user.Name,
		"role":      user.Role,
		"email":     user.Email,
		"ops_id":    user.OpsID,
		"exp":       time.Now().Add(12 * time.Hour).Unix(),
	})
	if err != nil {
		return
	}
	ctx.SetCookie(&http.Cookie{
		Name:     "soc5_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int((12 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("APP_ENV") == "production",
	})
}

func readSessionClaims(ctx echo.Context) (map[string]interface{}, bool) {
	cookie, err := ctx.Cookie("soc5_token")
	if err != nil {
		return nil, false
	}
	return ReadSessionClaims(cookie.Value)
}

func toHTTPError(err error) error {
	if appErr, ok := err.(AppError); ok {
		return echo.NewHTTPError(appErr.Code, appErr.Message)
	}
	return echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
}
