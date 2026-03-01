package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go_framework/internal/db"
	"go_framework/plugins/node/models"

	"gorm.io/gorm"
)

var (
	ErrInvalidProxyAuthType   = errors.New("invalid proxy auth_type")
	ErrInvalidProxyCredential = errors.New("invalid proxy credentials for auth_type")
)

// NodeProxyService handles CRUD operations for node proxies.
type NodeProxyService struct {
	db *gorm.DB
}

func NewNodeProxyService(db *gorm.DB) *NodeProxyService {
	return &NodeProxyService{db: db}
}

func NewNodeProxyServiceFromDefault() (*NodeProxyService, error) {
	gdb, err := db.GetGormDB()
	if err != nil {
		return nil, err
	}
	return NewNodeProxyService(gdb), nil
}

// List returns all proxies with optional active filter.
func (s *NodeProxyService) List(activeOnly bool) ([]models.NodeProxy, error) {
	var proxies []models.NodeProxy
	if activeOnly {
		if err := s.db.Where("is_active = ?", true).Find(&proxies).Error; err != nil {
			return nil, err
		}
		return proxies, nil
	}

	if err := s.db.Find(&proxies).Error; err != nil {
		return nil, err
	}
	return proxies, nil
}

// Get returns a proxy by ID.
func (s *NodeProxyService) Get(id string) (*models.NodeProxy, error) {
	var p models.NodeProxy
	if err := s.db.First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// Create inserts a new proxy. Application should set ID before calling.
func (s *NodeProxyService) Create(p *models.NodeProxy) error {
	if strings.TrimSpace(p.AuthType) == "" {
		p.AuthType = "api_key"
	}
	if err := validateProxyAuthConfig(p); err != nil {
		return err
	}
	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = time.Now().UTC()
	return s.db.Create(p).Error
}

// Update modifies an existing proxy.
func (s *NodeProxyService) Update(id string, changes map[string]interface{}) (*models.NodeProxy, error) {
	var p models.NodeProxy
	if err := s.db.First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}

	candidate := p
	if v, ok := changes["auth_type"]; ok {
		if sVal, ok := v.(string); ok {
			candidate.AuthType = sVal
		}
	}
	if v, ok := changes["api_user"]; ok {
		if sVal, ok := v.(string); ok {
			candidate.APIUser = &sVal
		}
	}
	if v, ok := changes["api_password"]; ok {
		if sVal, ok := v.(string); ok {
			candidate.APIPassword = &sVal
		}
	}
	if v, ok := changes["api_token"]; ok {
		if sVal, ok := v.(string); ok {
			candidate.APIToken = &sVal
		}
	}
	if err := validateProxyAuthConfig(&candidate); err != nil {
		return nil, err
	}

	changes["updated_at"] = time.Now().UTC()
	if err := s.db.Model(&p).Updates(changes).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Delete removes a proxy. It will set proxy_id to NULL on nodes via FK ON DELETE SET NULL.
func (s *NodeProxyService) Delete(id string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.NodeProxy{}, "id = ?", id).Error; err != nil {
			return err
		}
		return nil
	})
}

// ToggleActive flips is_active.
func (s *NodeProxyService) ToggleActive(id string, active bool) error {
	return s.db.Model(&models.NodeProxy{}).Where("id = ?", id).Update("is_active", active).Error
}

// AssignProxy assigns a proxy to a node (sets proxy_id). nodeID can be nil to unassign.
func (s *NodeProxyService) AssignProxy(nodeID string, proxyID *string) error {
	if proxyID != nil {
		// ensure proxy exists and is active
		var p models.NodeProxy
		if err := s.db.Where("id = ? AND is_active = ?", *proxyID, true).First(&p).Error; err != nil {
			return err
		}
	}
	// update node
	updates := map[string]interface{}{"proxy_id": proxyID, "updated_at": time.Now().UTC()}
	return s.db.Model(&models.Node{}).Where("id = ?", nodeID).Updates(updates).Error
}

func validateProxyAuthConfig(p *models.NodeProxy) error {
	authType := strings.TrimSpace(p.AuthType)
	switch authType {
	case "user_password":
		if !hasNonEmpty(p.APIUser) || !hasNonEmpty(p.APIPassword) {
			return fmt.Errorf("%w: auth_type=user_password requires api_user and api_password", ErrInvalidProxyCredential)
		}
	case "api_key":
		if !hasNonEmpty(p.APIToken) {
			return fmt.Errorf("%w: auth_type=api_key requires api_token", ErrInvalidProxyCredential)
		}
	default:
		return fmt.Errorf("%w: %s", ErrInvalidProxyAuthType, authType)
	}
	return nil
}

func hasNonEmpty(v *string) bool {
	return v != nil && strings.TrimSpace(*v) != ""
}
