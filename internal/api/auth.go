package api

import (
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
		// Safe fallback for local dev log a warning
		println("WARNING: JWT_SECRET not set, using insecure default. Set it before exposing to internet.")
		s = "change-me-set-JWT_SECRET-env-var"
	}
	return s
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func handleLogin(c *gin.Context) {
	var req loginRequest
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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": req.Username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	signed, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not sign token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": signed})
}

func handleSetupUser(c *gin.Context) {
	cfg := config.Get()
	if cfg.Auth.Username != "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "user already configured"})
		return
	}

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	c.JSON(http.StatusOK, gin.H{"message": "user created"})
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
	if err := config.Save(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
