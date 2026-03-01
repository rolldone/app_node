package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go_framework/plugins/node/models"
	"go_framework/plugins/node/services"
)

// Customer handlers for /api routes

type customerCreateContainerReq struct {
	TemplateID   string            `json:"template_id" binding:"required,uuid"`
	NodeID       *string           `json:"node_id" binding:"omitempty,uuid"`
	Subdomain    *string           `json:"subdomain"`
	InternalPort *int              `json:"internal_port" binding:"omitempty,min=1,max=65535"`
	RamMB        int               `json:"ram_mb" binding:"required,gt=0"`
	CPUPercent   int               `json:"cpu_percent" binding:"required,gt=0"`
	EnvVars      map[string]string `json:"env_vars"`
}

type customerUpdateContainerReq struct {
	Subdomain  *string           `json:"subdomain"`
	RamMB      *int              `json:"ram_mb" binding:"omitempty,gt=0"`
	CPUPercent *int              `json:"cpu_percent" binding:"omitempty,gt=0"`
	EnvVars    map[string]string `json:"env_vars"`
}

// GET /api/templates
func CustomerListTemplates(c *gin.Context) {
	var activeOnly *bool
	if raw := c.Query("is_active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid is_active"})
			return
		}
		activeOnly = &v
	}

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	limit := 10
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		limit = v
	}

	offset := 0
	if raw := c.Query("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
		offset = v
	}

	rows, total, err := svc.ListAppTemplates(activeOnly, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": rows, "total_count": total, "limit": limit, "offset": offset})
}

// GET /api/containers - list customer's own containers
func CustomerListContainers(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)
	if customerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid customer_id"})
		return
	}

	nodeID := c.Query("node_id")
	templateID := c.Query("template_id")
	status := c.Query("status")

	limit := 10
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		limit = v
	}

	offset := 0
	if raw := c.Query("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
		offset = v
	}

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	rows, total, err := svc.ListContainers(customerID, nodeID, templateID, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]gin.H, 0, len(rows))
	for i := range rows {
		resp = append(resp, containerResponse(&rows[i]))
	}
	c.JSON(http.StatusOK, gin.H{"containers": resp, "total_count": total, "limit": limit, "offset": offset})
}

// POST /api/containers - create container for customer
func CustomerCreateContainer(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)
	if customerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid customer_id"})
		return
	}

	var req customerCreateContainerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	row := &models.Container{
		CustomerID:   customerID,
		NodeID:       req.NodeID,
		TemplateID:   &req.TemplateID,
		Subdomain:    req.Subdomain,
		InternalPort: req.InternalPort,
		RamMB:        req.RamMB,
		CPUPercent:   req.CPUPercent,
	}

	if err := svc.CreateContainer(row, req.EnvVars); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"container": containerResponse(row)})
}

// GET /api/containers/:id - get customer's own container
func CustomerGetContainer(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)

	id := c.Param("id")
	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	row, err := svc.GetContainerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if row.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"container": containerResponse(row)})
}

// PUT /api/containers/:id - update customer's own container
func CustomerUpdateContainer(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)

	id := c.Param("id")
	var req customerUpdateContainerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	row, err := svc.GetContainerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if row.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if req.Subdomain != nil {
		row.Subdomain = req.Subdomain
	}
	if req.RamMB != nil {
		row.RamMB = *req.RamMB
	}
	if req.CPUPercent != nil {
		row.CPUPercent = *req.CPUPercent
	}

	if err := svc.UpdateContainer(row, req.EnvVars); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"container": containerResponse(row)})
}

// DELETE /api/containers/:id - delete customer's own container
func CustomerDeleteContainer(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)

	id := c.Param("id")
	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	row, err := svc.GetContainerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}

	if row.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := svc.DeleteContainer(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/containers/:id/deploy - deploy customer's own container
func CustomerDeployContainer(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)

	id := c.Param("id")

	var req deployContainerReq
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	// Verify ownership first
	container, err := svc.GetContainerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}
	if container.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	row, err := svc.DeployContainer(id, req.RegionCode)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrContainerNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		case errors.Is(err, services.ErrTemplateNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": "template not found"})
		case errors.Is(err, services.ErrTemplateInactive):
			c.JSON(http.StatusBadRequest, gin.H{"error": "template is inactive"})
		case errors.Is(err, services.ErrNodeNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": "node not found"})
		case errors.Is(err, services.ErrNoEligibleNode):
			c.JSON(http.StatusConflict, gin.H{"error": "no eligible node found"})
		case errors.Is(err, services.ErrInvalidState):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrDeployRequest):
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"container": containerResponse(row)})
}

// POST /api/containers/:id/reconcile - reconcile customer's own container
func CustomerReconcileContainer(c *gin.Context) {
	customerIDVal, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	customerID, _ := customerIDVal.(string)

	id := c.Param("id")
	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	// Verify ownership first
	container, err := svc.GetContainerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}
	if container.CustomerID != customerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	row, err := svc.ReconcileContainer(id)
	if err != nil {
		if errors.Is(err, services.ErrContainerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"container": containerResponse(row)})
}
