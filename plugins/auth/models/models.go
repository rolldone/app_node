package models

import (
	"time"

	internaluuid "go_framework/internal/uuid"

	"gorm.io/gorm"
)

type Admin struct {
	ID           string     `gorm:"type:uuid;primaryKey" json:"id"`
	Username     string     `gorm:"size:100;unique;not null" json:"username"`
	Email        string     `gorm:"size:255;unique;not null" json:"email"`
	PasswordHash string     `gorm:"type:text;not null" json:"-"`
	Level        string     `gorm:"type:admin_level;default:'STAFF'" json:"level"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Admin) TableName() string { return "admins" }

func (a *Admin) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		id, err := internaluuid.New()
		if err != nil {
			return err
		}
		a.ID = id
	}
	return nil
}

type AdminSession struct {
	ID               string     `gorm:"type:uuid;primaryKey" json:"id"`
	AdminID          string     `gorm:"type:uuid;index" json:"admin_id"`
	RefreshTokenHash string     `gorm:"type:text;not null" json:"-"`
	UserAgent        *string    `json:"user_agent"`
	IPAddress        *string    `json:"ip_address"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	ExpiresAt        *time.Time `json:"expires_at"`
	Revoked          bool       `gorm:"default:false" json:"revoked"`
}

func (AdminSession) TableName() string { return "admin_sessions" }

func (a *AdminSession) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		id, err := internaluuid.New()
		if err != nil {
			return err
		}
		a.ID = id
	}
	return nil
}

type AdminAPIKey struct {
	ID        string     `gorm:"type:uuid;primaryKey" json:"id"`
	AdminID   string     `gorm:"type:uuid;index" json:"admin_id"`
	Name      string     `gorm:"size:255" json:"name"`
	KeyHash   string     `gorm:"type:text;not null" json:"-"`
	Scopes    string     `gorm:"type:jsonb" json:"scopes"`
	IsActive  bool       `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}

func (AdminAPIKey) TableName() string { return "admin_api_keys" }

func (a *AdminAPIKey) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		id, err := internaluuid.New()
		if err != nil {
			return err
		}
		a.ID = id
	}
	return nil
}

type AdminAuditLog struct {
	ID         string    `gorm:"type:uuid;primaryKey" json:"id"`
	AdminID    *string   `gorm:"type:uuid" json:"admin_id"`
	Action     string    `gorm:"size:100;not null" json:"action"`
	TargetType *string   `gorm:"size:100" json:"target_type"`
	TargetID   *string   `gorm:"type:uuid" json:"target_id"`
	Meta       string    `gorm:"type:jsonb" json:"meta"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AdminAuditLog) TableName() string { return "admin_audit_logs" }

func (a *AdminAuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		id, err := internaluuid.New()
		if err != nil {
			return err
		}
		a.ID = id
	}
	return nil
}

type AdminPasswordReset struct {
	ID        string     `gorm:"type:uuid;primaryKey" json:"id"`
	AdminID   string     `gorm:"type:uuid;index" json:"admin_id"`
	TokenHash string     `gorm:"type:text;not null" json:"-"`
	ExpiresAt *time.Time `json:"expires_at"`
	Used      bool       `gorm:"default:false" json:"used"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (AdminPasswordReset) TableName() string { return "admin_password_resets" }

func (a *AdminPasswordReset) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		id, err := internaluuid.New()
		if err != nil {
			return err
		}
		a.ID = id
	}
	return nil
}
