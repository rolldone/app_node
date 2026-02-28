package services

import (
	"errors"
	"time"

	"go_framework/plugins/auth/models"

	"gorm.io/gorm"
)

// MemberService is a thin wrapper exposing customer/member-scoped methods.
type MemberService struct {
	core *AuthService
}

func NewMemberService(gdb *gorm.DB) (*MemberService, error) {
	if gdb == nil {
		return nil, errors.New("db is nil")
	}
	return &MemberService{core: New(gdb)}, nil
}

func (s *MemberService) CreateCustomer(cust *models.Customer) error {
	return s.core.CreateCustomer(cust)
}
func (s *MemberService) GetCustomerByEmail(email string) (*models.Customer, error) {
	return s.core.GetCustomerByEmail(email)
}
func (s *MemberService) GetCustomerByID(id string) (*models.Customer, error) {
	return s.core.GetCustomerByID(id)
}
func (s *MemberService) CustomerAuthenticateAndCreateSession(email, password string) (string, time.Time, string, time.Time, string, error) {
	return s.core.CustomerAuthenticateAndCreateSession(email, password)
}
func (s *MemberService) CustomerRefreshTokens(refreshToken string) (string, time.Time, string, time.Time, string, error) {
	return s.core.CustomerRefreshTokens(refreshToken)
}
func (s *MemberService) RevokeCustomerByRefreshHash(hash string) error {
	return s.core.RevokeCustomerByRefreshHash(hash)
}
