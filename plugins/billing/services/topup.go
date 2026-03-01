package services

import (
	"errors"
	"fmt"
	"time"

	"go_framework/internal/db"
	"go_framework/plugins/billing/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrGatewayNotFound     = errors.New("payment gateway not found")
	ErrGatewayInactive     = errors.New("payment gateway is inactive")
	ErrTopupNotFound       = errors.New("topup request not found")
	ErrInvalidAmount       = errors.New("amount is below minimum or above maximum")
	ErrDuplicateExternalID = errors.New("duplicate external_id detected")
	ErrTopupAlreadyPaid    = errors.New("topup already paid")
	ErrInvalidTopupStatus  = errors.New("invalid topup status for this operation")
)

type TopupService struct {
	db            *gorm.DB
	walletService *WalletService
}

func NewTopupService(gdb *gorm.DB) (*TopupService, error) {
	if gdb == nil {
		return nil, errors.New("db is nil")
	}
	walletSvc, err := NewWalletService(gdb)
	if err != nil {
		return nil, err
	}
	return &TopupService{db: gdb, walletService: walletSvc}, nil
}

func NewTopupServiceFromDefault() (*TopupService, error) {
	gdb, err := db.GetGormDB()
	if err != nil {
		return nil, err
	}
	return NewTopupService(gdb)
}

// CreateTopupRequest - Create new topup request
// Usage: Customer (own), Admin (for any customer)
func (s *TopupService) CreateTopupRequest(input struct {
	CustomerID string
	GatewayID  string
	Amount     float64
}) (*models.TopupRequest, error) {
	if input.Amount <= 0 {
		return nil, ErrNegativeAmount
	}

	// Verify gateway exists and active
	var gateway models.PaymentGateway
	if err := s.db.Where("id = ?", input.GatewayID).First(&gateway).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGatewayNotFound
		}
		return nil, err
	}

	if !gateway.IsActive {
		return nil, ErrGatewayInactive
	}

	// Validate amount range
	if input.Amount < gateway.MinAmount || input.Amount > gateway.MaxAmount {
		return nil, ErrInvalidAmount
	}

	// Calculate fee
	fee := (input.Amount * gateway.FeePercentage / 100) + gateway.FeeFixed
	totalPaid := input.Amount + fee

	// Create topup request
	topup := &models.TopupRequest{
		CustomerID: input.CustomerID,
		GatewayID:  input.GatewayID,
		Amount:     input.Amount,
		Fee:        fee,
		TotalPaid:  totalPaid,
		Status:     "PENDING",
	}

	// TODO: If gateway is AUTOMATIC, call payment gateway API to generate payment URL
	// For now, we'll leave payment_url and external_id empty (manual flow)

	if err := s.db.Create(topup).Error; err != nil {
		return nil, err
	}

	// Preload gateway info
	if err := s.db.Preload("Gateway").Where("id = ?", topup.ID).First(topup).Error; err != nil {
		return nil, err
	}

	return topup, nil
}

// GetTopupDetail - Get topup request detail
// Usage: Customer (own), Admin (any)
func (s *TopupService) GetTopupDetail(topupID string) (*models.TopupRequest, error) {
	var topup models.TopupRequest
	if err := s.db.Preload("Gateway").Where("id = ?", topupID).First(&topup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTopupNotFound
		}
		return nil, err
	}
	return &topup, nil
}

// ListTopupRequests - List topup requests
// Usage: Customer (own list), Admin (all list with filters)
func (s *TopupService) ListTopupRequests(filters struct {
	CustomerID *string
	GatewayID  *string
	Status     *string
	Limit      int
	Offset     int
}) ([]models.TopupRequest, int64, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := s.db.Model(&models.TopupRequest{}).Preload("Gateway")

	if filters.CustomerID != nil {
		query = query.Where("customer_id = ?", *filters.CustomerID)
	}
	if filters.GatewayID != nil {
		query = query.Where("gateway_id = ?", *filters.GatewayID)
	}
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var topups []models.TopupRequest
	if err := query.Order("created_at DESC").Limit(limit).Offset(filters.Offset).Find(&topups).Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// ProcessWebhook - Process payment gateway webhook (Idempotent)
// Usage: System/Public endpoint (called by payment gateway)
func (s *TopupService) ProcessWebhook(externalID string, webhookData map[string]interface{}) error {
	// This is a simplified version - actual implementation depends on gateway
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Find topup by external_id
		var topup models.TopupRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("external_id = ?", externalID).
			First(&topup).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTopupNotFound
			}
			return err
		}

		// Idempotency: if already SUCCESS, ignore
		if topup.Status == "SUCCESS" {
			return nil // Already processed
		}

		// Check if webhook indicates success payment
		// TODO: Parse webhook data based on gateway type
		status, ok := webhookData["status"].(string)
		if !ok {
			return errors.New("invalid webhook data: missing status")
		}

		// Update topup request
		now := time.Now()
		updateData := map[string]interface{}{
			"status":       status,
			"webhook_data": webhookData,
			"updated_at":   now,
		}

		if status == "SUCCESS" {
			updateData["paid_at"] = now
		}

		if err := tx.Model(&topup).Updates(updateData).Error; err != nil {
			return err
		}

		// If payment successful, credit wallet
		if status == "SUCCESS" {
			referenceID := topup.ID
			referenceType := "topup_request"
			_, err := s.walletService.RecordTransaction(tx, struct {
				CustomerID       string
				Amount           float64
				Type             string
				ReferenceID      *string
				ReferenceType    *string
				Description      string
				Metadata         string
				CreatedByAdminID *string
			}{
				CustomerID:       topup.CustomerID,
				Amount:           topup.Amount, // Only credit actual amount (not fee)
				Type:             "TOPUP",
				ReferenceID:      &referenceID,
				ReferenceType:    &referenceType,
				Description:      fmt.Sprintf("Top-up via %s", externalID),
				Metadata:         fmt.Sprintf(`{"topup_id": "%s", "gateway_id": "%s"}`, topup.ID, topup.GatewayID),
				CreatedByAdminID: nil,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

// ManualConfirmation - Admin manually confirm payment (for manual transfer)
// Usage: Admin only
func (s *TopupService) ManualConfirmation(adminID, topupID, notes string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var topup models.TopupRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", topupID).
			First(&topup).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTopupNotFound
			}
			return err
		}

		// Only PENDING can be confirmed
		if topup.Status != "PENDING" {
			return ErrInvalidTopupStatus
		}

		// Update status to SUCCESS
		now := time.Now()
		if err := tx.Model(&topup).Updates(map[string]interface{}{
			"status":     "SUCCESS",
			"paid_at":    now,
			"notes":      notes,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}

		// Credit wallet
		referenceID := topup.ID
		referenceType := "topup_request"
		_, err := s.walletService.RecordTransaction(tx, struct {
			CustomerID       string
			Amount           float64
			Type             string
			ReferenceID      *string
			ReferenceType    *string
			Description      string
			Metadata         string
			CreatedByAdminID *string
		}{
			CustomerID:       topup.CustomerID,
			Amount:           topup.Amount,
			Type:             "TOPUP",
			ReferenceID:      &referenceID,
			ReferenceType:    &referenceType,
			Description:      fmt.Sprintf("Manual top-up confirmation (Admin: %s)", adminID),
			Metadata:         fmt.Sprintf(`{"topup_id": "%s", "confirmed_by": "%s", "notes": "%s"}`, topup.ID, adminID, notes),
			CreatedByAdminID: &adminID,
		})

		return err
	})
}

// CancelTopup - Cancel pending topup
// Usage: Customer (own PENDING), Admin (any PENDING)
func (s *TopupService) CancelTopup(topupID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var topup models.TopupRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", topupID).
			First(&topup).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTopupNotFound
			}
			return err
		}

		// Only PENDING can be cancelled
		if topup.Status != "PENDING" {
			return ErrInvalidTopupStatus
		}

		return tx.Model(&topup).Updates(map[string]interface{}{
			"status":     "CANCELLED",
			"updated_at": time.Now(),
		}).Error
	})
}

// ListPaymentGateways - List available payment gateways
// Usage: Customer (active only), Admin (all)
func (s *TopupService) ListPaymentGateways(activeOnly bool) ([]models.PaymentGateway, error) {
	query := s.db.Model(&models.PaymentGateway{})
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	var gateways []models.PaymentGateway
	if err := query.Order("display_order ASC, name ASC").Find(&gateways).Error; err != nil {
		return nil, err
	}

	return gateways, nil
}
