package node

import (
	"go_framework/internal/plugins"
	pluginhandlers "go_framework/plugins/node/handlers"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// Plugin Node Management provides a CRUD sample scaffold.
type Plugin struct{}

func New() plugins.Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "node" }

func (p *Plugin) RegisterServices(db *gorm.DB) error { return nil }

func (p *Plugin) RegisterMiddleware() []plugins.MiddlewareDescriptor { return nil }

func (p *Plugin) RegisterRoutes(router *gin.Engine, admin *gin.RouterGroup, api *gin.RouterGroup, db *gorm.DB) error {
	// Admin routes - manage all resources
	admin.GET("/node/nodes", pluginhandlers.ListNodes)
	admin.POST("/node/nodes", pluginhandlers.CreateNode)
	admin.GET("/node/nodes/:id", pluginhandlers.GetNode)
	admin.PUT("/node/nodes/:id", pluginhandlers.UpdateNode)
	admin.DELETE("/node/nodes/:id", pluginhandlers.DeleteNode)
	admin.GET("/node/select", pluginhandlers.SelectBestNode)
	admin.GET("/node/templates", pluginhandlers.ListAppTemplates)
	admin.POST("/node/templates", pluginhandlers.CreateAppTemplate)
	admin.GET("/node/templates/:id", pluginhandlers.GetAppTemplate)
	admin.PUT("/node/templates/:id", pluginhandlers.UpdateAppTemplate)
	admin.DELETE("/node/templates/:id", pluginhandlers.DeleteAppTemplate)
	admin.GET("/node/containers", pluginhandlers.ListContainers)
	admin.POST("/node/containers", pluginhandlers.CreateContainer)
	admin.GET("/node/containers/:id", pluginhandlers.GetContainer)
	admin.PUT("/node/containers/:id", pluginhandlers.UpdateContainer)
	admin.DELETE("/node/containers/:id", pluginhandlers.DeleteContainer)
	admin.POST("/node/containers/:id/deploy", pluginhandlers.DeployContainer)
	admin.POST("/node/containers/:id/reconcile", pluginhandlers.ReconcileContainer)

	// Node proxy management (admin)
	admin.GET("/node/proxies", pluginhandlers.ListProxies)
	admin.POST("/node/proxies", pluginhandlers.CreateProxy)
	admin.GET("/node/proxies/:id", pluginhandlers.GetProxy)
	admin.PUT("/node/proxies/:id", pluginhandlers.UpdateProxy)
	admin.DELETE("/node/proxies/:id", pluginhandlers.DeleteProxy)
	admin.POST("/node/proxies/:id/toggle", pluginhandlers.ToggleProxy)

	// Assign/unassign proxy to node
	admin.PUT("/node/nodes/:id/proxy", pluginhandlers.AssignProxyToNode)

	// Customer API routes - manage own resources only
	if api != nil {
		api.GET("/templates", pluginhandlers.CustomerListTemplates)
		api.GET("/containers", pluginhandlers.CustomerListContainers)
		api.POST("/containers", pluginhandlers.CustomerCreateContainer)
		api.GET("/containers/:id", pluginhandlers.CustomerGetContainer)
		api.PUT("/containers/:id", pluginhandlers.CustomerUpdateContainer)
		api.DELETE("/containers/:id", pluginhandlers.CustomerDeleteContainer)
		api.POST("/containers/:id/deploy", pluginhandlers.CustomerDeployContainer)
		api.POST("/containers/:id/reconcile", pluginhandlers.CustomerReconcileContainer)
	}

	return nil
}

func (p *Plugin) Seed(db *gorm.DB) error { return nil }

func (p *Plugin) ConsoleCommands() []*cobra.Command {

	cmd := &cobra.Command{
		Use:   "node:hello",
		Short: "hello from node",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("hello from plugin node\\n")
		},
	}
	return []*cobra.Command{cmd}
}
