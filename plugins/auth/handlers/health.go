package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go_framework/internal/db"
)

// HealthHandler returns a health response for the plugin and verifies DB connectivity.
func HealthHandler(c *gin.Context) {
	// Try to obtain a gorm DB and report connectivity
	if gdb, err := db.GetGormDB(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "plugin": "auth", "db": false, "error": err.Error()})
		return
	} else if gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "plugin": "auth", "db": false, "error": "db connection is nil"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "plugin": "auth", "db": true})
}
