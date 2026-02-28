package services

import (
	"errors"
	"time"

	"go_framework/plugins/auth/models"

	"gorm.io/gorm"
)

// AdminService is a thin wrapper exposing admin-scoped methods.
type AdminService struct {
	core *AuthService
}

func NewAdminService(gdb *gorm.DB) (*AdminService, error) {
	if gdb == nil {
		return nil, errors.New("db is nil")
	}
	return &AdminService{core: New(gdb)}, nil
}

func (s *AdminService) CreateAdmin(a *models.Admin) error { return s.core.CreateAdmin(a) }
func (s *AdminService) GetAdminByEmail(email string) (*models.Admin, error) {
	return s.core.GetAdminByEmail(email)
}
func (s *AdminService) GetAdminByID(id string) (*models.Admin, error) { return s.core.GetAdminByID(id) }
func (s *AdminService) AuthenticateAndCreateSession(email, password string) (string, time.Time, string, time.Time, string, error) {
	return s.core.AuthenticateAndCreateSession(email, password)
}
func (s *AdminService) RefreshTokens(refreshToken string) (string, time.Time, string, time.Time, string, error) {
	return s.core.RefreshTokens(refreshToken)
}
func (s *AdminService) RevokeByRefreshHash(hash string) error {
	return s.core.RevokeByRefreshHash(hash)
}

func (s *AdminService) ListAdmins() ([]models.Admin, error) { return s.core.ListAdmins() }
func (s *AdminService) UpdateAdmin(a *models.Admin) error   { return s.core.UpdateAdmin(a) }
func (s *AdminService) DeleteAdmin(id string) error         { return s.core.DeleteAdmin(id) }
