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
			Target:   "store",
			Priority: 55,
			Handler:  MemberClaimsMiddleware(),
		},
	}
}

func (p *Plugin) RegisterRoutes(router *gin.Engine, admin *gin.RouterGroup, front *gin.RouterGroup, svcs *adminservices.AdminServices) error {
	admin.GET("/plugins/auth/health", pluginhandlers.HealthHandler)
	admin.POST("/admin/login", pluginhandlers.LoginHandler)
	admin.POST("/admin/refresh", pluginhandlers.RefreshHandler)
	admin.POST("/admin/logout", pluginhandlers.LogoutHandler)
	admin.GET("/admin/me", pluginhandlers.MeHandler)
	admin.POST("/admin/register", pluginhandlers.RegisterAdminHandler)
	// Admin CRUD
	admin.GET("/admin/list", pluginhandlers.ListAdminsHandler)
	admin.GET("/admin/:id", pluginhandlers.GetAdminHandler)
	admin.PUT("/admin/:id", pluginhandlers.UpdateAdminHandler)
	admin.DELETE("/admin/:id", pluginhandlers.DeleteAdminHandler)
	// member (customer) routes mounted on front router under /member
	if front != nil {
		front.POST("/member/register", pluginhandlers.MemberRegisterHandler)
		front.POST("/member/login", pluginhandlers.MemberLoginHandler)
		front.POST("/member/refresh", pluginhandlers.MemberRefreshHandler)
		front.POST("/member/logout", pluginhandlers.MemberLogoutHandler)
		front.GET("/member/me", pluginhandlers.MemberMeHandler)
	}
	return nil
}

func (p *Plugin) Seed(svcs *adminservices.AdminServices) error { return nil }
