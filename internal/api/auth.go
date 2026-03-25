package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"

	"stalkarr/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(mustGetJWTSecret())

func mustGetJWTSecret() string {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		println("WARNING: JWT_SECRET not set, using insecure default. Set it before exposing to internet.")
		return "insecure-default-secret-change-me"
	}
	return s
}

const (
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 7 * 24 * time.Hour
	refreshCookieName    = "stalkarr_refresh"
)

func generateAccessToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(accessTokenDuration).Unix(),
		"iat": time.Now().Unix(),
	})
	return token.SignedString(jwtSecret)
}

func generateRefreshToken() (raw string, hashed string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	raw = hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hashed = hex.EncodeToString(sum[:])
	return
}

func setRefreshCookie(c *gin.Context, token string) {
	c.SetCookie(
		refreshCookieName,
		token,
		int(refreshTokenDuration.Seconds()),
		"/",
		"",
		false,
		true,
	)
}

func clearRefreshCookie(c *gin.Context) {
	c.SetCookie(refreshCookieName, "", -1, "/", "", false, true)
}

func handleSetupUser(c *gin.Context) {
	cfg := config.Get()
	if cfg.Auth.Username != "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "already configured"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
		return
	}

	cfg.Auth.Username = req.Username
	cfg.Auth.PasswordHash = string(hash)
	if err := config.Save(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
		return
	}

	ip := c.ClientIP()

	cfg := config.Get()
	if cfg.Auth.Username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no user configured"})
		return
	}

	usernameOk := req.Username == cfg.Auth.Username
	passwordOk := usernameOk && bcrypt.CompareHashAndPassword([]byte(cfg.Auth.PasswordHash), []byte(req.Password)) == nil

	if !usernameOk || !passwordOk {
		locked, remaining := recordFailure(ip)
		if locked {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":            fmt.Sprintf("Too many failed attempts. Try again in %d minute(s).", int(remaining.Minutes())+1),
				"locked_out":       true,
				"retry_after_secs": int(remaining.Seconds()),
			})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	clearFailures(ip)

	accessToken, err := generateAccessToken(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
		return
	}

	rawRefresh, hashedRefresh, err := generateRefreshToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate refresh token"})
		return
	}

	// Store hash in config
	cfg.Auth.RefreshTokenHash = hashedRefresh
	if err := config.Save(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save session"})
		return
	}

	setRefreshCookie(c, rawRefresh)
	c.JSON(http.StatusOK, gin.H{"token": accessToken})
}

func handleRefresh(c *gin.Context) {
	rawToken, err := c.Cookie(refreshCookieName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no refresh token"})
		return
	}

	sum := sha256.Sum256([]byte(rawToken))
	incomingHash := hex.EncodeToString(sum[:])

	cfg := config.Get()
	if cfg.Auth.RefreshTokenHash == "" || incomingHash != cfg.Auth.RefreshTokenHash {
		clearRefreshCookie(c)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}

	accessToken, err := generateAccessToken(cfg.Auth.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": accessToken})
}

func handleLogout(c *gin.Context) {
	cfg := config.Get()
	cfg.Auth.RefreshTokenHash = ""
	config.Save(cfg)
	clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func handleChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if len(req.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new password must be at least 8 characters"})
		return
	}

	cfg := config.Get()
	if err := bcrypt.CompareHashAndPassword([]byte(cfg.Auth.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "current password is incorrect"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
		return
	}

	cfg.Auth.PasswordHash = string(hash)
	cfg.Auth.RefreshTokenHash = ""
	if err := config.Save(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save config"})
		return
	}

	clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func handleSetupStatus(c *gin.Context) {
	cfg := config.Get()
	c.JSON(http.StatusOK, gin.H{"configured": cfg.Auth.Username != ""})
}
