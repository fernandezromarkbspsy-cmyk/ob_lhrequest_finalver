package qr

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	qrcode "github.com/skip2/go-qrcode"
)

type Controller struct{}

func NewController() Controller {
	return Controller{}
}

func (c Controller) DriverAPI(ctx echo.Context) error {
	driverID := strings.TrimSpace(ctx.QueryParam("value"))
	if driverID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Driver ID is required")
	}
	png, err := qrcode.Encode(driverID, qrcode.Medium, 320)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to generate QR code")
	}
	return ctx.Blob(http.StatusOK, "image/png", png)
}
