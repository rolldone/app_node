package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go_framework/plugins/billing/models"
	"go_framework/plugins/billing/services"
)

// Admin handlers for /admin/billing routes

type adminCreateTopupReq struct {
	CustomerID string  `json:"customer_id" binding:"required,uuid"`
	GatewayID  string  `json:"gateway_id" binding:"required,uuid"`
	Amount     float64 `json:"amount" binding:"required,gt=0"`
}

type adminAdjustBalanceReq struct {
	CustomerID string  `json:"customer_id" binding:"required,uuid"`
	Amount     float64 `json:"amount" binding:"required"`
	Reason     string  `json:"reason" binding:"required"`
}

type adminConfirmTopupReq struct {
	Notes string `json:"notes" binding:"required"`
}

type adminRefundReq struct {
	CustomerID    string  `json:"customer_id" binding:"required,uuid"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
	ReferenceID   string  `json:"reference_id" binding:"required,uuid"`
	ReferenceType string  `json:"reference_type" binding:"required"`
	Reason        string  `json:"reason" binding:"required"`
}

type createGatewayReq struct {
	Name          string  `json:"name" binding:"required"`
	Slug          string  `json:"slug" binding:"required"`
	GatewayType   string  `json:"gateway_type" binding:"required,oneof=AUTOMATIC MANUAL"`
	IsActive      *bool   `json:"is_active"`
	Config        any     `json:"config"`
	FeePercentage float64 `json:"fee_percentage" binding:"gte=0"`
	FeeFixed      float64 `json:"fee_fixed" binding:"gte=0"`
	MinAmount     float64 `json:"min_amount" binding:"gt=0"`
	MaxAmount     float64 `json:"max_amount" binding:"gt=0"`
	DisplayOrder  int     `json:"display_order"`
}

type updateGatewayReq struct {
	Name          string   `json:"name" binding:"omitempty"`
	Slug          string   `json:"slug" binding:"omitempty"`
	GatewayType   string   `json:"gateway_type" binding:"omitempty,oneof=AUTOMATIC MANUAL"`
	IsActive      *bool    `json:"is_active"`
	Config        any      `json:"config"`
	FeePercentage *float64 `json:"fee_percentage" binding:"omitempty,gte=0"`
	FeeFixed      *float64 `json:"fee_fixed" binding:"omitempty,gte=0"`
	MinAmount     *float64 `json:"min_amount" binding:"omitempty,gt=0"`
	MaxAmount     *float64 `json:"max_amount" binding:"omitempty,gt=0"`
	DisplayOrder  *int     `json:"display_order"`
}

// ========== WALLET & TRANSACTIONS ==========

// GET /admin/billing/balance/:customer_id - Get customer balance
func AdminGetCustomerBalance(c *gin.Context) {
	customerID := c.Param("customer_id")

	svc, err := services.NewWalletServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	balance, err := svc.GetBalance(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"customer_id": customerID,
		"balance":     balance,
	})
}

// GET /admin/billing/transactions - Get all transactions with filter
func AdminGetAllTransactions(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	customerID := c.Query("customer_id")
	txnType := c.Query("type")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	svc, err := services.NewWalletServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	var customerIDPtr, typePtr, startPtr, endPtr *string
	if customerID != "" {
		customerIDPtr = &customerID
	}
	if txnType != "" {
		typePtr = &txnType
	}
	if startDate != "" {
		startPtr = &startDate
	}
	if endDate != "" {
		endPtr = &endDate
	}

	transactions, total, err := svc.GetAllTransactions(struct {
		CustomerID *string
		Type       *string
		StartDate  *string
		EndDate    *string
		Limit      int
		Offset     int
	}{
		CustomerID: customerIDPtr,
		Type:       typePtr,
		StartDate:  startPtr,
		EndDate:    endPtr,
		Limit:      limit,
		Offset:     offset,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// POST /admin/billing/adjust - Manual balance adjustment
func AdminAdjustBalance(c *gin.Context) {
	adminIDVal, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	adminID, _ := adminIDVal.(string)

	var req adminAdjustBalanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewWalletServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	transaction, err := svc.AdjustBalance(adminID, req.CustomerID, req.Amount, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transaction": transaction})
}

// ========== TOPUP MANAGEMENT ==========

// GET /admin/billing/topups - List all topup requests
func AdminListTopups(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	customerID := c.Query("customer_id")
	gatewayID := c.Query("gateway_id")
	status := c.Query("status")

	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	var customerIDPtr, gatewayIDPtr, statusPtr *string
	if customerID != "" {
		customerIDPtr = &customerID
	}
	if gatewayID != "" {
		gatewayIDPtr = &gatewayID
	}
	if status != "" {
		statusPtr = &status
	}

	topups, total, err := svc.ListTopupRequests(struct {
		CustomerID *string
		GatewayID  *string
		Status     *string
		Limit      int
		Offset     int
	}{
		CustomerID: customerIDPtr,
		GatewayID:  gatewayIDPtr,
		Status:     statusPtr,
		Limit:      limit,
		Offset:     offset,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"topups": topups,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GET /admin/billing/topups/:id - Get topup detail
func AdminGetTopup(c *gin.Context) {
	topupID := c.Param("id")

	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	topup, err := svc.GetTopupDetail(topupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"topup": topup})
}

// POST /admin/billing/topups - Create topup for any customer
func AdminCreateTopup(c *gin.Context) {
	var req adminCreateTopupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	topup, err := svc.CreateTopupRequest(struct {
		CustomerID string
		GatewayID  string
		Amount     float64
	}{
		CustomerID: req.CustomerID,
		GatewayID:  req.GatewayID,
		Amount:     req.Amount,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"topup": topup})
}

// POST /admin/billing/topups/:id/confirm - Manual confirmation
func AdminConfirmTopup(c *gin.Context) {
	adminIDVal, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	adminID, _ := adminIDVal.(string)

	topupID := c.Param("id")

	var req adminConfirmTopupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	if err := svc.ManualConfirmation(adminID, topupID, req.Notes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "topup confirmed"})
}

// DELETE /admin/billing/topups/:id - Cancel topup
func AdminCancelTopup(c *gin.Context) {
	topupID := c.Param("id")

	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	if err := svc.CancelTopup(topupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "topup cancelled"})
}

// ========== REFUND ==========

// POST /admin/billing/refund - Manual refund
func AdminRefund(c *gin.Context) {
	adminIDVal, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	adminID, _ := adminIDVal.(string)

	var req adminRefundReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewPurchaseServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	if err := svc.RefundBalance(struct {
		CustomerID    string
		Amount        float64
		ReferenceID   string
		ReferenceType string
		Reason        string
		AdminID       *string
	}{
		CustomerID:    req.CustomerID,
		Amount:        req.Amount,
		ReferenceID:   req.ReferenceID,
		ReferenceType: req.ReferenceType,
		Reason:        req.Reason,
		AdminID:       &adminID,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "refund processed"})
}

// ========== PAYMENT GATEWAY MANAGEMENT ==========

// GET /admin/billing/gateways - List all payment gateways
func AdminListGateways(c *gin.Context) {
	activeOnly := false
	if raw := c.Query("active_only"); raw == "true" {
		activeOnly = true
	}

	svc, err := services.NewGatewayServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	gateways, err := svc.ListGateways(activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"gateways": gateways})
}

// GET /admin/billing/gateways/:id - Get gateway detail
func AdminGetGateway(c *gin.Context) {
	gatewayID := c.Param("id")

	svc, err := services.NewGatewayServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	gateway, err := svc.GetGateway(gatewayID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"gateway": gateway})
}

// POST /admin/billing/gateways - Create payment gateway
func AdminCreateGateway(c *gin.Context) {
	var req createGatewayReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewGatewayServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	// Convert config to JSON string
	configJSON := "{}"
	if req.Config != nil {
		// Will be handled by GORM as JSONB
		configJSON = "" // Let GORM handle
	}

	gateway := &models.PaymentGateway{
		Name:          req.Name,
		Slug:          req.Slug,
		GatewayType:   req.GatewayType,
		IsActive:      req.IsActive != nil && *req.IsActive,
		Config:        configJSON,
		FeePercentage: req.FeePercentage,
		FeeFixed:      req.FeeFixed,
		MinAmount:     req.MinAmount,
		MaxAmount:     req.MaxAmount,
		DisplayOrder:  req.DisplayOrder,
	}

	if err := svc.CreateGateway(gateway); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"gateway": gateway})
}

// PUT /admin/billing/gateways/:id - Update payment gateway
func AdminUpdateGateway(c *gin.Context) {
	gatewayID := c.Param("id")

	var req updateGatewayReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewGatewayServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	// Get existing gateway
	gateway, err := svc.GetGateway(gatewayID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.Name != "" {
		gateway.Name = req.Name
	}
	if req.Slug != "" {
		gateway.Slug = req.Slug
	}
	if req.GatewayType != "" {
		gateway.GatewayType = req.GatewayType
	}
	if req.IsActive != nil {
		gateway.IsActive = *req.IsActive
	}
	if req.FeePercentage != nil {
		gateway.FeePercentage = *req.FeePercentage
	}
	if req.FeeFixed != nil {
		gateway.FeeFixed = *req.FeeFixed
	}
	if req.MinAmount != nil {
		gateway.MinAmount = *req.MinAmount
	}
	if req.MaxAmount != nil {
		gateway.MaxAmount = *req.MaxAmount
	}
	if req.DisplayOrder != nil {
		gateway.DisplayOrder = *req.DisplayOrder
	}

	if err := svc.UpdateGateway(gateway); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"gateway": gateway})
}

// DELETE /admin/billing/gateways/:id - Delete payment gateway
func AdminDeleteGateway(c *gin.Context) {
	gatewayID := c.Param("id")

	svc, err := services.NewGatewayServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	if err := svc.DeleteGateway(gatewayID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "gateway deleted"})
}

// PATCH /admin/billing/gateways/:id/toggle - Toggle gateway status
func AdminToggleGateway(c *gin.Context) {
	gatewayID := c.Param("id")

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewGatewayServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	if err := svc.ToggleGatewayStatus(gatewayID, req.IsActive); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "gateway status updated"})
}
