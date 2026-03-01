package services

import (
	"errors"

	"go_framework/internal/db"
	"go_framework/plugins/billing/models"

	"gorm.io/gorm"
)

type GatewayService struct {
	db *gorm.DB
}

func NewGatewayService(gdb *gorm.DB) (*GatewayService, error) {
	if gdb == nil {
		return nil, errors.New("db is nil")
	}
	return &GatewayService{db: gdb}, nil
}

func NewGatewayServiceFromDefault() (*GatewayService, error) {
	gdb, err := db.GetGormDB()
	if err != nil {
		return nil, err
	}
	return NewGatewayService(gdb)
}

// ListGateways - List payment gateways
// Usage: Admin (all), Customer via TopupService (active only)
func (s *GatewayService) ListGateways(activeOnly bool) ([]models.PaymentGateway, error) {
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

// GetGateway - Get gateway by ID
// Usage: Admin
func (s *GatewayService) GetGateway(id string) (*models.PaymentGateway, error) {
	var gateway models.PaymentGateway
	if err := s.db.Where("id = ?", id).First(&gateway).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGatewayNotFound
		}
		return nil, err
	}
	return &gateway, nil
}

// CreateGateway - Create new payment gateway
// Usage: Admin only
func (s *GatewayService) CreateGateway(input *models.PaymentGateway) error {
	return s.db.Create(input).Error
}

// UpdateGateway - Update payment gateway
// Usage: Admin only
func (s *GatewayService) UpdateGateway(input *models.PaymentGateway) error {
	return s.db.Save(input).Error
}

// DeleteGateway - Delete payment gateway
// Usage: Admin only
func (s *GatewayService) DeleteGateway(id string) error {
	// Check if gateway has topup requests
	var count int64
	if err := s.db.Model(&models.TopupRequest{}).Where("gateway_id = ?", id).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return errors.New("cannot delete gateway with existing topup requests")
	}

	return s.db.Delete(&models.PaymentGateway{}, "id = ?", id).Error
}

// ToggleGatewayStatus - Enable/disable gateway
// Usage: Admin only
func (s *GatewayService) ToggleGatewayStatus(id string, isActive bool) error {
	return s.db.Model(&models.PaymentGateway{}).
		Where("id = ?", id).
		Update("is_active", isActive).Error
}
