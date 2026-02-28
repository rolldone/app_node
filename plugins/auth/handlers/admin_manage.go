package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go_framework/internal/db"
	"go_framework/plugins/auth/models"
	"go_framework/plugins/auth/services"
)

type createAdminReq struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Level    string `json:"level" binding:"omitempty,oneof=STAFF SUPERADMIN"`
}

// POST /admin/register (create admin) - protected: SUPERADMIN only
func RegisterAdminHandler(c *gin.Context) {
	// require admin_level from middleware
	lvlv, ok := c.Get("admin_level")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing admin auth"})
		return
	}
	levelStr, _ := lvlv.(string)
	if levelStr != "SUPERADMIN" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
		return
	}

	var req createAdminReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	// hash password
	coreSvc := services.New(gdb)
	pwHash, err := coreSvc.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	admin := &models.Admin{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: pwHash,
		Level:        req.Level,
		IsActive:     true,
	}

	svc, serr := services.NewAdminService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	if err := svc.CreateAdmin(admin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": admin.ID})
}
