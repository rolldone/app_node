package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"go_framework/internal/db"
	"go_framework/internal/keydb"
	"go_framework/plugins/auth/services"
)

type adminMeResponse struct {
	User  interface{}  `json:"user"`
	Flash *keydb.Flash `json:"flash,omitempty"`
}

// GET /admin/me
// Returns the current admin user and any pending flash message (one-time read).
// Requires JWT token via Authorization header (injected by AdminClaimsMiddleware).
func AdminMeHandler(c *gin.Context) {
	// Get admin_id from context (set by AdminClaimsMiddleware)
	adminIDInterface, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	adminID, ok := adminIDInterface.(string)
	if !ok || adminID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get DB
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	// Get admin by ID
	svc, err := services.NewAdminService(gdb)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service error"})
		return
	}

	admin, err := svc.GetAdminByID(adminID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "admin not found"})
		return
	}

	// Get and clear flash from KeyDB (one-time read)
	// The session ID should be stored somewhere accessible or derived from JWT.
	// For now, use adminID as a simple key suffix; adjust if you have proper session tracking.
	ctx := context.Background()
	flash, _ := keydb.GetAndClearFlash(ctx, adminID) // ignore errors, flash is optional

	resp := adminMeResponse{
		User:  admin,
		Flash: flash,
	}

	c.JSON(http.StatusOK, resp)
}
