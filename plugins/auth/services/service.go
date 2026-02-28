package services

import (
	"errors"
	"os"
	"time"

	authpkg "go_framework/internal/auth"
	"go_framework/internal/db"
	"go_framework/plugins/auth/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db *gorm.DB
}

func New(gdb *gorm.DB) *AuthService {
	return &AuthService{db: gdb}
}

func NewFromDefault() (*AuthService, error) {
	gdb, err := db.GetGormDB()
	if err != nil {
		return nil, err
	}
	if gdb == nil {
		return nil, errors.New("db is nil")
	}
	return New(gdb), nil
}

func (s *AuthService) CreateAdmin(a *models.Admin) error {
	return s.db.Create(a).Error
}

func (s *AuthService) GetAdminByEmail(email string) (*models.Admin, error) {
	var a models.Admin
	if err := s.db.Where("email = ?", email).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *AuthService) GetAdminByID(id string) (*models.Admin, error) {
	var a models.Admin
	if err := s.db.Where("id = ?", id).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

// ListAdmins returns all admins (no pagination for now)
func (s *AuthService) ListAdmins() ([]models.Admin, error) {
	var list []models.Admin
	if err := s.db.Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateAdmin updates an existing admin record. Caller should set ID.
func (s *AuthService) UpdateAdmin(a *models.Admin) error {
	// Save will update based on primary key
	return s.db.Save(a).Error
}

// DeleteAdmin deletes an admin by id
func (s *AuthService) DeleteAdmin(id string) error {
	return s.db.Delete(&models.Admin{}, "id = ?", id).Error
}

func (s *AuthService) CreateSession(sess *models.AdminSession) error {
	return s.db.Create(sess).Error
}

func (s *AuthService) RevokeSession(id string) error {
	return s.db.Model(&models.AdminSession{}).Where("id = ?", id).Update("revoked", true).Error
}

// HashPassword returns bcrypt hash
func (s *AuthService) HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *AuthService) CheckPassword(hash, pw string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)); err != nil {
		return false
	}
	return true
}

var (
	defaultAccessTTL  = 15 * time.Minute
	defaultRefreshTTL = 7 * 24 * time.Hour
)

// AuthenticateAndCreateSession authenticates credentials and creates a refresh session
func (s *AuthService) AuthenticateAndCreateSession(email, password string) (accessToken string, accessExp time.Time, refreshPlain string, refreshExp time.Time, sessionID string, err error) {
	admin, err := s.GetAdminByEmail(email)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	if !s.CheckPassword(admin.PasswordHash, password) {
		return "", time.Time{}, "", time.Time{}, "", errors.New("invalid credentials")
	}
	if !admin.IsActive {
		return "", time.Time{}, "", time.Time{}, "", errors.New("account inactive")
	}

	// generate access token
	at, aexp, err := authpkg.GenerateAccessTokenWithLevel(admin.ID, admin.Level, defaultAccessTTL)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}

	// generate refresh token (plain + hash)
	plain, hash, err := authpkg.GenerateOpaqueRefreshToken()
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	rexpires := time.Now().Add(defaultRefreshTTL)

	sess := &models.AdminSession{
		AdminID:          admin.ID,
		RefreshTokenHash: hash,
		ExpiresAt:        &rexpires,
		Revoked:          false,
	}
	if err := s.CreateSession(sess); err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}

	return at, aexp, plain, rexpires, sess.ID, nil
}

// RefreshTokens consumes a refresh token and returns new tokens (rotating)
func (s *AuthService) RefreshTokens(refreshToken string) (accessToken string, accessExp time.Time, newRefreshPlain string, refreshExp time.Time, sessionID string, err error) {
	// hash incoming token
	hash := authpkg.HashOpaqueToken(refreshToken)
	var sess models.AdminSession
	if err := s.db.Where("refresh_token_hash = ?", hash).First(&sess).Error; err != nil {
		return "", time.Time{}, "", time.Time{}, "", errors.New("invalid refresh token")
	}
	if sess.Revoked {
		return "", time.Time{}, "", time.Time{}, "", errors.New("refresh token revoked")
	}
	if sess.ExpiresAt != nil && sess.ExpiresAt.Before(time.Now()) {
		return "", time.Time{}, "", time.Time{}, "", errors.New("refresh token expired")
	}

	// load admin
	admin, err := s.GetAdminByID(sess.AdminID)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}

	// revoke old session
	if err := s.RevokeSession(sess.ID); err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}

	// create new tokens
	at, aexp, err := authpkg.GenerateAccessTokenWithLevel(admin.ID, admin.Level, defaultAccessTTL)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	plain, newHash, err := authpkg.GenerateOpaqueRefreshToken()
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	rexpires := time.Now().Add(defaultRefreshTTL)
	newSess := &models.AdminSession{
		AdminID:          admin.ID,
		RefreshTokenHash: newHash,
		ExpiresAt:        &rexpires,
		Revoked:          false,
	}
	if err := s.CreateSession(newSess); err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}

	return at, aexp, plain, rexpires, newSess.ID, nil
}

// RevokeByRefreshHash revokes a session by its stored refresh token hash
func (s *AuthService) RevokeByRefreshHash(hash string) error {
	return s.db.Model(&models.AdminSession{}).Where("refresh_token_hash = ?", hash).Update("revoked", true).Error
}

// -- Customer (member) helpers --
func (s *AuthService) CreateCustomer(cust *models.Customer) error {
	return s.db.Create(cust).Error
}

func (s *AuthService) GetCustomerByEmail(email string) (*models.Customer, error) {
	var c models.Customer
	if err := s.db.Where("email = ?", email).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *AuthService) GetCustomerByID(id string) (*models.Customer, error) {
	var c models.Customer
	if err := s.db.Where("id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *AuthService) CreateCustomerSession(sess *models.CustomerSession) error {
	return s.db.Create(sess).Error
}

func (s *AuthService) RevokeCustomerByRefreshHash(hash string) error {
	return s.db.Model(&models.CustomerSession{}).Where("refresh_token_hash = ?", hash).Update("revoked", true).Error
}

// CustomerAuthenticateAndCreateSession authenticates customer and creates session
func (s *AuthService) CustomerAuthenticateAndCreateSession(email, password string) (accessToken string, accessExp time.Time, refreshPlain string, refreshExp time.Time, sessionID string, err error) {
	cust, err := s.GetCustomerByEmail(email)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	if !s.CheckPassword(cust.PasswordHash, password) {
		return "", time.Time{}, "", time.Time{}, "", errors.New("invalid credentials")
	}
	if !cust.IsActive {
		return "", time.Time{}, "", time.Time{}, "", errors.New("account inactive")
	}

	at, aexp, err := authpkg.GenerateAccessTokenWithLevel(cust.ID, "customer", defaultAccessTTL)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	plain, hash, err := authpkg.GenerateOpaqueRefreshToken()
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	rexpires := time.Now().Add(defaultRefreshTTL)
	sess := &models.CustomerSession{
		CustomerID:       cust.ID,
		RefreshTokenHash: hash,
		ExpiresAt:        &rexpires,
		Revoked:          false,
	}
	if err := s.CreateCustomerSession(sess); err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	return at, aexp, plain, rexpires, sess.ID, nil
}

// CustomerRefreshTokens rotates customer refresh tokens
func (s *AuthService) CustomerRefreshTokens(refreshToken string) (accessToken string, accessExp time.Time, newRefreshPlain string, refreshExp time.Time, sessionID string, err error) {
	hash := authpkg.HashOpaqueToken(refreshToken)
	var sess models.CustomerSession
	if err := s.db.Where("refresh_token_hash = ?", hash).First(&sess).Error; err != nil {
		return "", time.Time{}, "", time.Time{}, "", errors.New("invalid refresh token")
	}
	if sess.Revoked {
		return "", time.Time{}, "", time.Time{}, "", errors.New("refresh token revoked")
	}
	if sess.ExpiresAt != nil && sess.ExpiresAt.Before(time.Now()) {
		return "", time.Time{}, "", time.Time{}, "", errors.New("refresh token expired")
	}
	cust, err := s.GetCustomerByID(sess.CustomerID)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	if err := s.db.Model(&models.CustomerSession{}).Where("id = ?", sess.ID).Update("revoked", true).Error; err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	at, aexp, err := authpkg.GenerateAccessTokenWithLevel(cust.ID, "customer", defaultAccessTTL)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	plain, newHash, err := authpkg.GenerateOpaqueRefreshToken()
	if err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	rexpires := time.Now().Add(defaultRefreshTTL)
	newSess := &models.CustomerSession{
		CustomerID:       cust.ID,
		RefreshTokenHash: newHash,
		ExpiresAt:        &rexpires,
		Revoked:          false,
	}
	if err := s.CreateCustomerSession(newSess); err != nil {
		return "", time.Time{}, "", time.Time{}, "", err
	}
	return at, aexp, plain, rexpires, newSess.ID, nil
}

// Helper to get env-based TTL overrides (optional)
func init() {
	if v := os.Getenv("AUTH_ACCESS_TTL_MIN"); v != "" {
		// ignore parse errors; keep default
	}
}
