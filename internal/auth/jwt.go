package auth

import (
        "errors"
        "log"
        "net/http"
        "os"
        "time"

        "github.com/golang-jwt/jwt/v5"
        "github.com/labstack/echo/v4"
)

const CookieName = "soc5_session"

type Claims struct {
        UserID uint   `json:"uid"`
        Role   string `json:"role"`
        Name   string `json:"name"`
        jwt.RegisteredClaims
}

type SessionUser struct {
        ID   uint
        Role string
        Name string
}

func secret() []byte {
        s := os.Getenv("APP_SECRET")
        if s == "" {
                log.Fatal("APP_SECRET environment variable must be set")
        }
        if len(s) < 32 {
                log.Fatal("APP_SECRET must be at least 32 characters")
        }
        return []byte(s)
}

func IssueToken(userID uint, role, name string) (string, error) {
        claims := Claims{
                UserID: userID,
                Role:   role,
                Name:   name,
                RegisteredClaims: jwt.RegisteredClaims{
                        ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
                        IssuedAt:  jwt.NewNumericDate(time.Now()),
                },
        }
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        return token.SignedString(secret())
}

func ParseToken(tokenStr string) (*Claims, error) {
        token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
                if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                        return nil, errors.New("unexpected signing method")
                }
                return secret(), nil
        })
        if err != nil {
                return nil, err
        }
        claims, ok := token.Claims.(*Claims)
        if !ok || !token.Valid {
                return nil, errors.New("invalid token")
        }
        return claims, nil
}

func SetSessionCookie(c echo.Context, tokenStr string) {
        cookie := new(http.Cookie)
        cookie.Name = CookieName
        cookie.Value = tokenStr
        cookie.Path = "/"
        cookie.HttpOnly = true
        cookie.SameSite = http.SameSiteStrictMode
        cookie.MaxAge = 12 * 60 * 60
        c.SetCookie(cookie)
}

func ClearSessionCookie(c echo.Context) {
        cookie := new(http.Cookie)
        cookie.Name = CookieName
        cookie.Value = ""
        cookie.Path = "/"
        cookie.HttpOnly = true
        cookie.SameSite = http.SameSiteStrictMode
        cookie.MaxAge = -1
        c.SetCookie(cookie)
}

func GetSessionUser(c echo.Context) *SessionUser {
        cookie, err := c.Cookie(CookieName)
        if err != nil {
                return nil
        }
        claims, err := ParseToken(cookie.Value)
        if err != nil {
                return nil
        }
        return &SessionUser{ID: claims.UserID, Role: claims.Role, Name: claims.Name}
}

func RequireAuth() echo.MiddlewareFunc {
        return func(next echo.HandlerFunc) echo.HandlerFunc {
                return func(c echo.Context) error {
                        user := GetSessionUser(c)
                        if user == nil {
                                return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
                        }
                        c.Set("auth_user", user)
                        return next(c)
                }
        }
}
