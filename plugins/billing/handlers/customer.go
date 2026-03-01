package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go_framework/plugins/billing/services"
)

// Customer handlers for /api routes

type createTopupReq struct {
	GatewayID string  `json:"gateway_id" binding:"required,uuid"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
}

// GET /api/billing/balance - Get customer wallet balance
func CustomerGetBalance(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)
	if customerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid customer_id"})
		return
	}

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

// GET /api/billing/transactions - Get customer transaction history
func CustomerGetTransactions(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)
	if customerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid customer_id"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	svc, err := services.NewWalletServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	transactions, total, err := svc.GetTransactionHistory(customerID, limit, offset)
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

// GET /api/billing/gateways - List active payment gateways
func CustomerListGateways(c *gin.Context) {
	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	gateways, err := svc.ListPaymentGateways(true) // active only
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"gateways": gateways})
}

// POST /api/billing/topup - Create topup request
func CustomerCreateTopup(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)
	if customerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid customer_id"})
		return
	}

	var req createTopupReq
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
		CustomerID: customerID,
		GatewayID:  req.GatewayID,
		Amount:     req.Amount,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"topup": topup})
}

// GET /api/billing/topup - List customer's topup requests
func CustomerListTopups(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)
	if customerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid customer_id"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.Query("status")

	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	var statusPtr *string
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
		CustomerID: &customerID,
		GatewayID:  nil,
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

// GET /api/billing/topup/:id - Get topup detail
func CustomerGetTopup(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)

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

	// Ownership check
	if topup.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"topup": topup})
}

// DELETE /api/billing/topup/:id - Cancel pending topup
func CustomerCancelTopup(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)

	topupID := c.Param("id")

	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	// Verify ownership first
	topup, err := svc.GetTopupDetail(topupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if topup.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := svc.CancelTopup(topupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "topup cancelled"})
}
