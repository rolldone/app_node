package services

import (
	"errors"
	"fmt"

	"go_framework/internal/db"
	"go_framework/plugins/billing/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrCustomerNotFound    = errors.New("customer not found")
	ErrNegativeAmount      = errors.New("amount must be positive")
	ErrInvalidBalance      = errors.New("invalid balance state")
)

type WalletService struct {
	db *gorm.DB
}

func NewWalletService(gdb *gorm.DB) (*WalletService, error) {
	if gdb == nil {
		return nil, errors.New("db is nil")
	}
	return &WalletService{db: gdb}, nil
}

func NewWalletServiceFromDefault() (*WalletService, error) {
	gdb, err := db.GetGormDB()
	if err != nil {
		return nil, err
	}
	return NewWalletService(gdb)
}

// GetBalance - Get customer wallet balance
// Usage: Customer (own), Admin (any customer)
func (s *WalletService) GetBalance(customerID string) (float64, error) {
	var customer models.CustomerBalance
	if err := s.db.Table("customers").Where("id = ?", customerID).First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrCustomerNotFound
		}
		return 0, err
	}
	return customer.WalletBalance, nil
}

// GetTransactionHistory - Get wallet transaction history
// Usage: Customer (own history), Admin (any customer history)
// Params: customerID, limit, offset
func (s *WalletService) GetTransactionHistory(customerID string, limit, offset int) ([]models.WalletTransaction, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var transactions []models.WalletTransaction
	var total int64

	query := s.db.Model(&models.WalletTransaction{}).Where("customer_id = ?", customerID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// RecordTransaction - Internal helper to record wallet mutation
// Usage: Internal only (called from topup/purchase services)
// MUST be called within a transaction context
func (s *WalletService) RecordTransaction(tx *gorm.DB, input struct {
	CustomerID       string
	Amount           float64 // Positive for credit, negative for debit
	Type             string  // TOPUP, PURCHASE, REFUND, etc
	ReferenceID      *string
	ReferenceType    *string
	Description      string
	Metadata         string
	CreatedByAdminID *string
}) (*models.WalletTransaction, error) {
	if input.Amount == 0 {
		return nil, errors.New("amount cannot be zero")
	}

	// Get current balance (with row lock)
	var customer models.CustomerBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Table("customers").
		Where("id = ?", input.CustomerID).
		First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	balanceBefore := customer.WalletBalance
	balanceAfter := balanceBefore + input.Amount

	// Validate balance tidak negatif (kecuali allowed)
	if balanceAfter < 0 {
		return nil, ErrInsufficientBalance
	}

	// Update customer balance
	if err := tx.Table("customers").
		Where("id = ?", input.CustomerID).
		Update("wallet_balance", balanceAfter).Error; err != nil {
		return nil, err
	}

	// Create transaction record
	transaction := &models.WalletTransaction{
		CustomerID:       input.CustomerID,
		Amount:           input.Amount,
		BalanceBefore:    balanceBefore,
		BalanceAfter:     balanceAfter,
		Type:             input.Type,
		ReferenceID:      input.ReferenceID,
		ReferenceType:    input.ReferenceType,
		Description:      input.Description,
		Metadata:         input.Metadata,
		CreatedByAdminID: input.CreatedByAdminID,
	}

	if err := tx.Create(transaction).Error; err != nil {
		return nil, err
	}

	return transaction, nil
}

// AdjustBalance - Manual balance adjustment by admin
// Usage: Admin only (for correction, compensation, etc)
func (s *WalletService) AdjustBalance(adminID, customerID string, amount float64, reason string) (*models.WalletTransaction, error) {
	if amount == 0 {
		return nil, ErrNegativeAmount
	}

	var transaction *models.WalletTransaction
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txn, err := s.RecordTransaction(tx, struct {
			CustomerID       string
			Amount           float64
			Type             string
			ReferenceID      *string
			ReferenceType    *string
			Description      string
			Metadata         string
			CreatedByAdminID *string
		}{
			CustomerID:       customerID,
			Amount:           amount,
			Type:             "ADMIN_ADJUSTMENT",
			ReferenceID:      nil,
			ReferenceType:    nil,
			Description:      fmt.Sprintf("Admin adjustment: %s", reason),
			Metadata:         fmt.Sprintf(`{"reason": "%s", "admin_id": "%s"}`, reason, adminID),
			CreatedByAdminID: &adminID,
		})
		if err != nil {
			return err
		}
		transaction = txn
		return nil
	})

	if err != nil {
		return nil, err
	}

	return transaction, nil
}

// GetAllTransactions - Get all transactions with filter (Admin only)
// Usage: Admin only - for billing dashboard/report
func (s *WalletService) GetAllTransactions(filters struct {
	CustomerID *string
	Type       *string
	StartDate  *string
	EndDate    *string
	Limit      int
	Offset     int
}) ([]models.WalletTransaction, int64, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	query := s.db.Model(&models.WalletTransaction{})

	if filters.CustomerID != nil {
		query = query.Where("customer_id = ?", *filters.CustomerID)
	}
	if filters.Type != nil {
		query = query.Where("type = ?", *filters.Type)
	}
	if filters.StartDate != nil {
		query = query.Where("created_at >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("created_at <= ?", *filters.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var transactions []models.WalletTransaction
	if err := query.Order("created_at DESC").Limit(limit).Offset(filters.Offset).Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}
