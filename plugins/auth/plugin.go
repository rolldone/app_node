package auth

import (
	adminservices "go_framework/internal/admin/services"
	"go_framework/internal/plugins"
	pluginhandlers "go_framework/plugins/auth/handlers"

	"github.com/gin-gonic/gin"
)

// Plugin auth provides a minimal scaffold.
type Plugin struct{}

// New returns a new plugin instance.
func New() plugins.Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "auth" }

func (p *Plugin) RegisterServices(svcs *adminservices.AdminServices) error { return nil }

func (p *Plugin) RegisterMiddleware() []plugins.MiddlewareDescriptor {
	return []plugins.MiddlewareDescriptor{
		{
			Name:     "plugins.auth.claims",
			Target:   "admin",
			Priority: 55,
			Handler:  AdminClaimsMiddleware(),
		},
		{
			Name:     "plugins.auth.member_claims",
			Target:   "api",
			Priority: 55,
			Handler:  MemberClaimsMiddleware(),
		},
	}
}

func (p *Plugin) RegisterRoutes(router *gin.Engine, admin *gin.RouterGroup, api *gin.RouterGroup, svcs *adminservices.AdminServices) error {
	admin.GET("/plugins/auth/health", pluginhandlers.HealthHandler)

	// Admin auth endpoints on /admin/auth
	authAdmin := admin.Group("/auth")
	authAdmin.POST("/login", pluginhandlers.LoginHandler)
	authAdmin.POST("/refresh", pluginhandlers.RefreshHandler)
	authAdmin.POST("/logout", pluginhandlers.LogoutHandler)
	authAdmin.GET("/me", pluginhandlers.MeHandler)
	authAdmin.POST("/register", pluginhandlers.RegisterAdminHandler)
	authAdmin.GET("/list", pluginhandlers.ListAdminsHandler)
	authAdmin.GET("/:id", pluginhandlers.GetAdminHandler)
	authAdmin.PUT("/:id", pluginhandlers.UpdateAdminHandler)
	authAdmin.DELETE("/:id", pluginhandlers.DeleteAdminHandler)

	// Admin customer management at /admin/customers
	adminCustomers := admin.Group("/customers")
	adminCustomers.GET("", pluginhandlers.ListCustomersHandler)
	adminCustomers.POST("", pluginhandlers.CreateCustomerHandler)
	adminCustomers.GET("/:id", pluginhandlers.GetCustomerHandler)
	adminCustomers.PUT("/:id", pluginhandlers.UpdateCustomerHandler)
	adminCustomers.DELETE("/:id", pluginhandlers.DeleteCustomerHandler)

	// Customer (member) auth routes on /api/auth
	if api != nil {
		api.POST("/auth/register", pluginhandlers.MemberRegisterHandler)
		api.POST("/auth/login", pluginhandlers.MemberLoginHandler)
		api.POST("/auth/refresh", pluginhandlers.MemberRefreshHandler)
		api.POST("/auth/logout", pluginhandlers.MemberLogoutHandler)
		api.GET("/auth/me", pluginhandlers.MemberMeHandler)
	}
	return nil
}

func (p *Plugin) Seed(svcs *adminservices.AdminServices) error { return nil }
