package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go_framework/plugins/billing/services"
)

// Webhook handlers for payment gateway callbacks (public endpoint)

// POST /webhooks/midtrans - Midtrans payment notification
func WebhookMidtrans(c *gin.Context) {
	// TODO: Validate Midtrans signature
	// serverKey := "your-server-key"
	// signatureSent := c.GetHeader("X-Signature")
	// Verify signature before processing

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Extract order_id (our external_id)
	orderID, ok := payload["order_id"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing order_id"})
		return
	}

	// Map Midtrans transaction_status to our status
	transactionStatus, _ := payload["transaction_status"].(string)
	fraudStatus, _ := payload["fraud_status"].(string)

	var status string
	switch transactionStatus {
	case "capture":
		if fraudStatus == "accept" {
			status = "SUCCESS"
		} else {
			status = "PENDING"
		}
	case "settlement":
		status = "SUCCESS"
	case "pending":
		status = "PENDING"
	case "deny", "cancel", "expire":
		status = "FAILED"
	default:
		status = "PENDING"
	}

	payload["status"] = status

	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	if err := svc.ProcessWebhook(orderID, payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// POST /webhooks/xendit - Xendit payment notification
func WebhookXendit(c *gin.Context) {
	// TODO: Validate Xendit callback token
	// callbackToken := c.GetHeader("X-Callback-Token")
	// Verify token before processing

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Extract external_id
	externalID, ok := payload["external_id"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing external_id"})
		return
	}

	// Map Xendit status to our status
	xenditStatus, _ := payload["status"].(string)

	var status string
	switch xenditStatus {
	case "PAID":
		status = "SUCCESS"
	case "PENDING":
		status = "PENDING"
	case "EXPIRED", "FAILED":
		status = "FAILED"
	default:
		status = "PENDING"
	}

	payload["status"] = status

	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	if err := svc.ProcessWebhook(externalID, payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Generic webhook handler - can be used for testing
// POST /webhooks/payment - Generic payment webhook
func WebhookGeneric(c *gin.Context) {
	var payload struct {
		ExternalID string                 `json:"external_id" binding:"required"`
		Status     string                 `json:"status" binding:"required"`
		Data       map[string]interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Merge status into data
	if payload.Data == nil {
		payload.Data = make(map[string]interface{})
	}
	payload.Data["status"] = payload.Status

	svc, err := services.NewTopupServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	if err := svc.ProcessWebhook(payload.ExternalID, payload.Data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
