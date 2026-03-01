package models

import (
	"time"

	appuuid "go_framework/internal/uuid"

	"gorm.io/gorm"
)

// PaymentGateway represents payment provider configuration
type PaymentGateway struct {
	ID            string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name          string    `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Slug          string    `gorm:"size:50;not null;uniqueIndex" json:"slug"`
	GatewayType   string    `gorm:"size:50;not null" json:"gateway_type"` // AUTOMATIC, MANUAL
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	Config        string    `gorm:"type:jsonb" json:"config"`
	FeePercentage float64   `gorm:"type:decimal(5,2);default:0.00" json:"fee_percentage"`
	FeeFixed      float64   `gorm:"type:decimal(15,2);default:0.00" json:"fee_fixed"`
	MinAmount     float64   `gorm:"type:decimal(15,2);default:10000.00" json:"min_amount"`
	MaxAmount     float64   `gorm:"type:decimal(15,2);default:10000000.00" json:"max_amount"`
	DisplayOrder  int       `gorm:"default:0" json:"display_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (PaymentGateway) TableName() string { return "payment_gateways" }

func (g *PaymentGateway) BeforeCreate(tx *gorm.DB) error {
	if g.ID == "" {
		id, err := appuuid.New()
		if err != nil {
			return err
		}
		g.ID = id
	}
	return nil
}

// TopupRequest tracks customer top-up requests
type TopupRequest struct {
	ID             string     `gorm:"type:uuid;primaryKey" json:"id"`
	CustomerID     string     `gorm:"type:uuid;not null;index" json:"customer_id"`
	GatewayID      string     `gorm:"type:uuid;not null;index" json:"gateway_id"`
	Amount         float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	Fee            float64    `gorm:"type:decimal(15,2);default:0.00" json:"fee"`
	TotalPaid      float64    `gorm:"type:decimal(15,2);not null" json:"total_paid"`
	ExternalID     *string    `gorm:"size:255;uniqueIndex" json:"external_id,omitempty"`
	PaymentURL     *string    `gorm:"type:text" json:"payment_url,omitempty"`
	PaymentMethod  *string    `gorm:"size:100" json:"payment_method,omitempty"`
	PaymentChannel *string    `gorm:"size:100" json:"payment_channel,omitempty"`
	Status         string     `gorm:"type:topup_status;not null;default:PENDING;index" json:"status"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	ExpiredAt      *time.Time `json:"expired_at,omitempty"`
	WebhookData    string     `gorm:"type:jsonb" json:"webhook_data,omitempty"`
	Notes          *string    `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt      time.Time  `gorm:"index:idx_topup_created" json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relations
	Gateway *PaymentGateway `gorm:"foreignKey:GatewayID" json:"gateway,omitempty"`
}

func (TopupRequest) TableName() string { return "topup_requests" }

func (t *TopupRequest) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		id, err := appuuid.New()
		if err != nil {
			return err
		}
		t.ID = id
	}
	return nil
}

// WalletTransaction represents audit trail for all wallet mutations
type WalletTransaction struct {
	ID               string    `gorm:"type:uuid;primaryKey" json:"id"`
	CustomerID       string    `gorm:"type:uuid;not null;index" json:"customer_id"`
	Amount           float64   `gorm:"type:decimal(15,2);not null" json:"amount"` // Positive (credit), Negative (debit)
	BalanceBefore    float64   `gorm:"type:decimal(15,2);not null" json:"balance_before"`
	BalanceAfter     float64   `gorm:"type:decimal(15,2);not null" json:"balance_after"`
	Type             string    `gorm:"type:transaction_type;not null;index" json:"type"` // TOPUP, PURCHASE, REFUND, etc
	ReferenceID      *string   `gorm:"type:uuid;index" json:"reference_id,omitempty"`
	ReferenceType    *string   `gorm:"size:100" json:"reference_type,omitempty"` // topup_request, container, manual
	Description      string    `gorm:"type:text" json:"description"`
	Metadata         string    `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedByAdminID *string   `gorm:"type:uuid" json:"created_by_admin_id,omitempty"`
	CreatedAt        time.Time `gorm:"index:idx_wallet_txn_created" json:"created_at"`
}

func (WalletTransaction) TableName() string { return "wallet_transactions" }

func (t *WalletTransaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		id, err := appuuid.New()
		if err != nil {
			return err
		}
		t.ID = id
	}
	return nil
}

// Customer extension - we need to reference wallet_balance
// This is just for reference, actual Customer model is in auth plugin
type CustomerBalance struct {
	ID            string  `gorm:"type:uuid;primaryKey" json:"id"`
	WalletBalance float64 `gorm:"type:decimal(15,2);default:0.00;not null" json:"wallet_balance"`
}

func (CustomerBalance) TableName() string { return "customers" }
