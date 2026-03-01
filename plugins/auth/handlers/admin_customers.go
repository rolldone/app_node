package handlers

import (
	"net/http"
	"strconv"

	"go_framework/internal/db"
	"go_framework/plugins/auth/models"
	"go_framework/plugins/auth/services"

	"github.com/gin-gonic/gin"
)

type createCustomerReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"omitempty"`
	IsActive *bool  `json:"is_active" binding:"omitempty"`
}

type updateCustomerReq struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"omitempty,min=8"`
	FullName string `json:"full_name" binding:"omitempty"`
	IsActive *bool  `json:"is_active" binding:"omitempty"`
}

// GET /admin/customers
func ListCustomersHandler(c *gin.Context) {
	// Create response with fields matching other list endpoints (like nodes)
	type CustomerListResponse struct {
		Customers  []models.Customer `json:"customers"`
		TotalCount int64             `json:"total_count"`
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
	svc, serr := services.NewMemberService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	list, total, err := svc.ListCustomersWithPagination(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, CustomerListResponse{
		Customers:  list,
		TotalCount: total,
	})
}

// GET /admin/customers/:id
func GetCustomerHandler(c *gin.Context) {
	id := c.Param("id")
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
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"customer": cust})
}

// POST /admin/customers  (SUPERADMIN only)
func CreateCustomerHandler(c *gin.Context) {
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

	var req createCustomerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	authCore := services.New(gdb)
	memberSvc, serr := services.NewMemberService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	ph, err := authCore.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	cust := &models.Customer{
		Email:        req.Email,
		PasswordHash: ph,
		FullName:     req.FullName,
	}
	if req.IsActive != nil {
		cust.IsActive = *req.IsActive
	}
	if err := memberSvc.CreateCustomer(cust); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": cust.ID})
}

// PUT /admin/customers/:id  (STAFF and SUPERADMIN)
func UpdateCustomerHandler(c *gin.Context) {
	lvlv, ok := c.Get("admin_level")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing admin auth"})
		return
	}
	levelStr, _ := lvlv.(string)
	if levelStr != "STAFF" && levelStr != "SUPERADMIN" {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient privileges"})
		return
	}

	id := c.Param("id")
	var req updateCustomerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	gdb, err := db.GetGormDB()
	if err != nil || gdb == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	authCore := services.New(gdb)
	memberSvc, serr := services.NewMemberService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	cust, err := memberSvc.GetCustomerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	if req.Email != "" {
		cust.Email = req.Email
	}
	if req.FullName != "" {
		cust.FullName = req.FullName
	}
	if req.Password != "" {
		ph, err := authCore.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		cust.PasswordHash = ph
	}
	if req.IsActive != nil {
		cust.IsActive = *req.IsActive
	}
	if err := memberSvc.UpdateCustomer(cust); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /admin/customers/:id  (SUPERADMIN only)
func DeleteCustomerHandler(c *gin.Context) {
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
	svc, serr := services.NewMemberService(gdb)
	if serr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": serr.Error()})
		return
	}
	if err := svc.DeleteCustomer(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
