package models

import (
	"time"

	internaluuid "go_framework/internal/uuid"

	"gorm.io/gorm"
)

type Customer struct {
	ID              string     `gorm:"type:uuid;primaryKey" json:"id"`
	Email           string     `gorm:"size:255;unique;not null" json:"email"`
	PasswordHash    string     `gorm:"type:text;not null" json:"-"`
	FullName        string     `gorm:"size:255" json:"full_name"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	Status          string     `gorm:"size:50;default:'ACTIVE'" json:"status"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Customer) TableName() string { return "customers" }

func (c *Customer) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		id, err := internaluuid.New()
		if err != nil {
			return err
		}
		c.ID = id
	}
	return nil
}

type CustomerSession struct {
	ID               string     `gorm:"type:uuid;primaryKey" json:"id"`
	CustomerID       string     `gorm:"type:uuid;index" json:"customer_id"`
	RefreshTokenHash string     `gorm:"type:text;not null" json:"-"`
	UserAgent        string     `json:"user_agent"`
	IPAddress        string     `json:"ip_address"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	ExpiresAt        *time.Time `json:"expires_at"`
	Revoked          bool       `gorm:"default:false" json:"revoked"`
}

func (CustomerSession) TableName() string { return "customer_sessions" }

func (c *CustomerSession) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		id, err := internaluuid.New()
		if err != nil {
			return err
		}
		c.ID = id
	}
	return nil
}
