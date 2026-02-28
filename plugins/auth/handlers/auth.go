package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	authpkg "go_framework/internal/auth"
	"go_framework/internal/db"
	"go_framework/plugins/auth/services"
)

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// POST /admin/login
func LoginHandler(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	svc, serr := services.NewAdminService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}

	at, aexp, refreshPlain, rexp, sid, err := svc.AuthenticateAndCreateSession(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":       at,
		"access_expires_at":  aexp.Format(time.RFC3339),
		"refresh_token":      refreshPlain,
		"refresh_expires_at": rexp.Format(time.RFC3339),
		"session_id":         sid,
	})
}

// POST /admin/refresh
func RefreshHandler(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	svc, serr := services.NewAdminService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}

	at, aexp, newRefresh, rexp, sid, err := svc.RefreshTokens(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":       at,
		"access_expires_at":  aexp.Format(time.RFC3339),
		"refresh_token":      newRefresh,
		"refresh_expires_at": rexp.Format(time.RFC3339),
		"session_id":         sid,
	})
}

// POST /admin/logout
func LogoutHandler(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	svc, serr := services.NewAdminService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	hash := authpkg.HashOpaqueToken(req.RefreshToken)
	if err := svc.RevokeByRefreshHash(hash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /admin/me
func MeHandler(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
		return
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
		return
	}
	tokenStr := parts[1]
	claims, err := authpkg.ParseAccessTokenClaims(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	svc, serr := services.NewAdminService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	admin, err := svc.GetAdminByID(claims.AdminID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"admin": admin})
}

// POST /admin/register (create admin) - protected: SUPERADMIN only
// RegisterAdminHandler moved to admin_manage.go
