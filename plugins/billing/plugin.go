package billing

import (
	"go_framework/internal/plugins"
	pluginhandlers "go_framework/plugins/billing/handlers"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// Plugin Billing & Financial Management provides a CRUD sample scaffold.
type Plugin struct{}

func New() plugins.Plugin { return &Plugin{} }

func (p *Plugin) ID() string { return "billing" }

func (p *Plugin) RegisterServices(db *gorm.DB) error { return nil }

func (p *Plugin) RegisterMiddleware() []plugins.MiddlewareDescriptor { return nil }

func (p *Plugin) RegisterRoutes(router *gin.Engine, admin *gin.RouterGroup, api *gin.RouterGroup, db *gorm.DB) error {
	// ========== ADMIN ROUTES (/admin/billing/*) ==========
	billing := admin.Group("/billing")
	{
		// Wallet & Transactions
		billing.GET("/balance/:customer_id", pluginhandlers.AdminGetCustomerBalance)
		billing.GET("/transactions", pluginhandlers.AdminGetAllTransactions)
		billing.POST("/adjust", pluginhandlers.AdminAdjustBalance)

		// Topup Management
		billing.GET("/topups", pluginhandlers.AdminListTopups)
		billing.GET("/topups/:id", pluginhandlers.AdminGetTopup)
		billing.POST("/topups", pluginhandlers.AdminCreateTopup)
		billing.POST("/topups/:id/confirm", pluginhandlers.AdminConfirmTopup)
		billing.DELETE("/topups/:id", pluginhandlers.AdminCancelTopup)

		// Refund
		billing.POST("/refund", pluginhandlers.AdminRefund)

		// Payment Gateway Management
		billing.GET("/gateways", pluginhandlers.AdminListGateways)
		billing.GET("/gateways/:id", pluginhandlers.AdminGetGateway)
		billing.POST("/gateways", pluginhandlers.AdminCreateGateway)
		billing.PUT("/gateways/:id", pluginhandlers.AdminUpdateGateway)
		billing.DELETE("/gateways/:id", pluginhandlers.AdminDeleteGateway)
		billing.PATCH("/gateways/:id/toggle", pluginhandlers.AdminToggleGateway)
	}

	// ========== CUSTOMER ROUTES (/api/billing/*) ==========
	customerBilling := api.Group("/billing")
	{
		// Wallet
		customerBilling.GET("/balance", pluginhandlers.CustomerGetBalance)
		customerBilling.GET("/transactions", pluginhandlers.CustomerGetTransactions)

		// Payment Gateways (active only)
		customerBilling.GET("/gateways", pluginhandlers.CustomerListGateways)

		// Topup
		customerBilling.GET("/topup", pluginhandlers.CustomerListTopups)
		customerBilling.GET("/topup/:id", pluginhandlers.CustomerGetTopup)
		customerBilling.POST("/topup", pluginhandlers.CustomerCreateTopup)
		customerBilling.DELETE("/topup/:id", pluginhandlers.CustomerCancelTopup)
	}

	// ========== PUBLIC WEBHOOK ROUTES (/webhooks/*) ==========
	// No authentication required - payment gateway callbacks
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("/midtrans", pluginhandlers.WebhookMidtrans)
		webhooks.POST("/xendit", pluginhandlers.WebhookXendit)
		webhooks.POST("/payment", pluginhandlers.WebhookGeneric) // Generic for testing
	}

	return nil
}

func (p *Plugin) Seed(db *gorm.DB) error { return nil }

func (p *Plugin) ConsoleCommands() []*cobra.Command {

	cmd := &cobra.Command{
		Use:   "billing:hello",
		Short: "hello from billing",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("hello from plugin billing\\n")
		},
	}
	return []*cobra.Command{cmd}
}
