package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	authpkg "go_framework/internal/auth"
	"go_framework/internal/db"
	"go_framework/plugins/auth/models"
	"go_framework/plugins/auth/services"
)

type memberRegisterReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name"`
}

type memberLoginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// POST /member/register
func MemberRegisterHandler(c *gin.Context) {
	var req memberRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	svc := services.New(gdb)

	pwHash, err := svc.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	cust := &models.Customer{}
	cust.Email = req.Email
	cust.PasswordHash = pwHash
	cust.FullName = req.FullName
	if err := svc.CreateCustomer(cust); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// TODO: send verification email
	c.JSON(http.StatusCreated, gin.H{"id": cust.ID})
}

// POST /member/login
func MemberLoginHandler(c *gin.Context) {
	var req memberLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	svc, serr := services.NewMemberService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}

	at, aexp, refreshPlain, rexp, sid, err := svc.CustomerAuthenticateAndCreateSession(req.Email, req.Password)
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

// POST /member/refresh
func MemberRefreshHandler(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	svc, serr := services.NewMemberService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	at, aexp, newRefresh, rexp, sid, err := svc.CustomerRefreshTokens(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": at, "access_expires_at": aexp.Format(time.RFC3339), "refresh_token": newRefresh, "refresh_expires_at": rexp.Format(time.RFC3339), "session_id": sid})
}

// POST /member/logout
func MemberLogoutHandler(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	svc, serr := services.NewMemberService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	hash := authpkg.HashOpaqueToken(req.RefreshToken)
	if err := svc.RevokeCustomerByRefreshHash(hash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /member/me
func MemberMeHandler(c *gin.Context) {
	// expects middleware to set `user_id` or `customer_id`
	idv, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	id, _ := idv.(string)
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	svc, serr := services.NewMemberService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	cust, err := svc.GetCustomerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"member": gin.H{"id": cust.ID, "email": cust.Email, "full_name": cust.FullName}})
}
