package services

import (
	"errors"
	"fmt"

	"go_framework/internal/db"

	"gorm.io/gorm"
)

type PurchaseService struct {
	db            *gorm.DB
	walletService *WalletService
}

func NewPurchaseService(gdb *gorm.DB) (*PurchaseService, error) {
	if gdb == nil {
		return nil, errors.New("db is nil")
	}
	walletSvc, err := NewWalletService(gdb)
	if err != nil {
		return nil, err
	}
	return &PurchaseService{db: gdb, walletService: walletSvc}, nil
}

func NewPurchaseServiceFromDefault() (*PurchaseService, error) {
	gdb, err := db.GetGormDB()
	if err != nil {
		return nil, err
	}
	return NewPurchaseService(gdb)
}

// ValidateBalance - Check if customer has enough balance
// Usage: Internal (called before purchase)
func (s *PurchaseService) ValidateBalance(customerID string, requiredAmount float64) (bool, float64, error) {
	balance, err := s.walletService.GetBalance(customerID)
	if err != nil {
		return false, 0, err
	}

	if balance < requiredAmount {
		return false, balance, nil
	}

	return true, balance, nil
}

// DeductBalance - Deduct balance for purchase (container deployment, addon, etc)
// Usage: Internal (called from node plugin deploy flow)
// MUST be called within transaction context
func (s *PurchaseService) DeductBalance(tx *gorm.DB, input struct {
	CustomerID    string
	Amount        float64
	ReferenceID   string // Container ID, addon ID, etc
	ReferenceType string // 'container', 'addon', 'domain', etc
	Description   string
	Metadata      string
}) error {
	if input.Amount <= 0 {
		return ErrNegativeAmount
	}

	// Deduct balance (negative amount)
	refID := input.ReferenceID
	refType := input.ReferenceType
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
		CustomerID:       input.CustomerID,
		Amount:           -input.Amount, // Negative for debit
		Type:             "PURCHASE",
		ReferenceID:      &refID,
		ReferenceType:    &refType,
		Description:      input.Description,
		Metadata:         input.Metadata,
		CreatedByAdminID: nil,
	})

	return err
}

// RefundBalance - Refund balance (on container delete, error, etc)
// Usage: Internal (called from node plugin delete/error flow), Admin (manual refund)
// MUST be called within transaction context if part of larger operation
func (s *PurchaseService) RefundBalance(input struct {
	CustomerID    string
	Amount        float64
	ReferenceID   string // Original container ID, etc
	ReferenceType string // 'container', 'addon', etc
	Reason        string
	AdminID       *string // If manual refund by admin
}) error {
	if input.Amount <= 0 {
		return ErrNegativeAmount
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		refID := input.ReferenceID
		refType := input.ReferenceType
		description := fmt.Sprintf("Refund: %s", input.Reason)

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
			CustomerID:       input.CustomerID,
			Amount:           input.Amount, // Positive for credit
			Type:             "REFUND",
			ReferenceID:      &refID,
			ReferenceType:    &refType,
			Description:      description,
			Metadata:         fmt.Sprintf(`{"reason": "%s", "original_reference": "%s"}`, input.Reason, input.ReferenceID),
			CreatedByAdminID: input.AdminID,
		})

		return err
	})

	return err
}

// CalculateContainerPrice - Calculate container deployment price
// Usage: Node plugin (before deploy), Customer (price preview)
// TODO: Implement pricing logic based on RAM, CPU, duration, etc
func (s *PurchaseService) CalculateContainerPrice(ramMB, cpuPercent int, durationHours int) (float64, error) {
	// Simplified pricing logic - customize based on business model
	// Example: Rp 500 per GB RAM per hour + Rp 100 per CPU % per hour

	if ramMB <= 0 || cpuPercent <= 0 || durationHours <= 0 {
		return 0, errors.New("invalid pricing parameters")
	}

	ramGB := float64(ramMB) / 1024.0
	pricePerGBHour := 500.0
	pricePerCPUHour := 100.0

	ramCost := ramGB * pricePerGBHour * float64(durationHours)
	cpuCost := float64(cpuPercent) * pricePerCPUHour * float64(durationHours)

	totalPrice := ramCost + cpuCost

	return totalPrice, nil
}

// GetPurchaseHistory - Get purchase history for customer
// Usage: Customer (own), Admin (any customer)
func (s *PurchaseService) GetPurchaseHistory(customerID string, limit, offset int) (interface{}, error) {
	// Reuse wallet service to get PURCHASE type transactions
	transactions, total, err := s.walletService.GetTransactionHistory(customerID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Filter only PURCHASE and REFUND types
	var purchaseHistory []interface{}
	for _, txn := range transactions {
		if txn.Type == "PURCHASE" || txn.Type == "REFUND" {
			purchaseHistory = append(purchaseHistory, map[string]interface{}{
				"id":             txn.ID,
				"amount":         txn.Amount,
				"balance_before": txn.BalanceBefore,
				"balance_after":  txn.BalanceAfter,
				"type":           txn.Type,
				"reference_id":   txn.ReferenceID,
				"reference_type": txn.ReferenceType,
				"description":    txn.Description,
				"created_at":     txn.CreatedAt,
			})
		}
	}

	return map[string]interface{}{
		"data":   purchaseHistory,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}, nil
}
