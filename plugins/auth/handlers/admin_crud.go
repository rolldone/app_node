package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go_framework/internal/db"
	"go_framework/plugins/auth/services"
)

type updateAdminReq struct {
	Username string `json:"username" binding:"omitempty"`
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"omitempty,min=8"`
	Level    string `json:"level" binding:"omitempty,oneof=STAFF SUPERADMIN"`
	IsActive *bool  `json:"is_active" binding:"omitempty"`
}

// GET /admin/list
func ListAdminsHandler(c *gin.Context) {
	type AdminListResponse struct {
		Admins     []map[string]interface{} `json:"admins"`
		TotalCount int64                    `json:"total_count"`
	}

	// Parse pagination
	limit := 10
	offset := 0

	if val := c.Query("limit"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if val := c.Query("offset"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			offset = parsed
		}
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
	list, total, err := svc.ListAdminsWithPagination(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to map to exclude password_hash
	var dtos []map[string]interface{}
	for _, admin := range list {
		dtos = append(dtos, map[string]interface{}{
			"id":            admin.ID,
			"username":      admin.Username,
			"email":         admin.Email,
			"level":         admin.Level,
			"is_active":     admin.IsActive,
			"last_login_at": admin.LastLoginAt,
			"created_at":    admin.CreatedAt,
			"updated_at":    admin.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, AdminListResponse{
		Admins:     dtos,
		TotalCount: total,
	})
}

// GET /admin/:id
func GetAdminHandler(c *gin.Context) {
	id := c.Param("id")
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
	admin, err := svc.GetAdminByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"admin": admin})
}

// PUT /admin/:id
func UpdateAdminHandler(c *gin.Context) {
	// require SUPERADMIN
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

	id := c.Param("id")
	var req updateAdminReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	core := services.New(gdb)
	svc, serr := services.NewAdminService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	admin, err := svc.GetAdminByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}
	if req.Username != "" {
		admin.Username = req.Username
	}
	if req.Email != "" {
		admin.Email = req.Email
	}
	if req.Level != "" {
		admin.Level = req.Level
	}
	if req.IsActive != nil {
		admin.IsActive = *req.IsActive
	}
	if req.Password != "" {
		h, err := core.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		admin.PasswordHash = h
	}
	if err := svc.UpdateAdmin(admin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"admin": admin})
}

// DELETE /admin/:id
func DeleteAdminHandler(c *gin.Context) {
	// require SUPERADMIN
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
	id := c.Param("id")
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
	if err := svc.DeleteAdmin(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
