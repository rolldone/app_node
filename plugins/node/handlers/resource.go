package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"go_framework/plugins/node/models"
	"go_framework/plugins/node/services"
)

type createNodeReq struct {
	Name        string `json:"name" binding:"required"`
	RegionCode  string `json:"region_code" binding:"required"`
	RegionName  string `json:"region_name" binding:"required"`
	APIEndpoint string `json:"api_endpoint" binding:"required,url"`
	APIKey      string `json:"api_key" binding:"required"`
	IPAddress   string `json:"ip_address" binding:"required"`
	MaxRamMB    int    `json:"max_ram_mb" binding:"required,gt=0"`
	UsedRamMB   int    `json:"used_ram_mb" binding:"gte=0"`
	Status      string `json:"status" binding:"omitempty,oneof=ACTIVE MAINTENANCE FULL"`
}

type updateNodeReq struct {
	Name        string `json:"name" binding:"omitempty"`
	RegionCode  string `json:"region_code" binding:"omitempty"`
	RegionName  string `json:"region_name" binding:"omitempty"`
	APIEndpoint string `json:"api_endpoint" binding:"omitempty,url"`
	APIKey      string `json:"api_key" binding:"omitempty"`
	IPAddress   string `json:"ip_address" binding:"omitempty"`
	MaxRamMB    *int   `json:"max_ram_mb" binding:"omitempty,gt=0"`
	UsedRamMB   *int   `json:"used_ram_mb" binding:"omitempty,gte=0"`
	Status      string `json:"status" binding:"omitempty,oneof=ACTIVE MAINTENANCE FULL"`
}

type createTemplateReq struct {
	AppName         string `json:"app_name" binding:"required"`
	DockerImage     string `json:"docker_image" binding:"required"`
	DefaultRamMB    int    `json:"default_ram_mb" binding:"omitempty,gt=0"`
	DefaultCPULimit int    `json:"default_cpu_limit" binding:"omitempty,gt=0"`
	ConfigContent   string `json:"config_content"`
	ConfigType      string `json:"config_type"`
	IsActive        *bool  `json:"is_active"`
}

type updateTemplateReq struct {
	AppName         string `json:"app_name" binding:"omitempty"`
	DockerImage     string `json:"docker_image" binding:"omitempty"`
	DefaultRamMB    *int   `json:"default_ram_mb" binding:"omitempty,gt=0"`
	DefaultCPULimit *int   `json:"default_cpu_limit" binding:"omitempty,gt=0"`
	ConfigContent   string `json:"config_content"`
	ConfigType      string `json:"config_type"`
	IsActive        *bool  `json:"is_active"`
}

type createContainerReq struct {
	CustomerID   string            `json:"customer_id" binding:"required,uuid"`
	NodeID       *string           `json:"node_id" binding:"omitempty,uuid"`
	TemplateID   *string           `json:"template_id" binding:"omitempty,uuid"`
	ExternalID   *string           `json:"external_id"`
	Subdomain    *string           `json:"subdomain"`
	InternalPort *int              `json:"internal_port" binding:"omitempty,min=1,max=65535"`
	RamMB        int               `json:"ram_mb" binding:"required,gt=0"`
	CPUPercent   int               `json:"cpu_percent" binding:"required,gt=0"`
	Status       string            `json:"status" binding:"omitempty,oneof=PENDING DEPLOYING RUNNING ERROR"`
	EnvVars      map[string]string `json:"env_vars"`
}

type updateContainerReq struct {
	NodeID       *string           `json:"node_id" binding:"omitempty,uuid"`
	TemplateID   *string           `json:"template_id" binding:"omitempty,uuid"`
	ExternalID   *string           `json:"external_id"`
	Subdomain    *string           `json:"subdomain"`
	InternalPort *int              `json:"internal_port" binding:"omitempty,min=1,max=65535"`
	RamMB        *int              `json:"ram_mb" binding:"omitempty,gt=0"`
	CPUPercent   *int              `json:"cpu_percent" binding:"omitempty,gt=0"`
	Status       string            `json:"status" binding:"omitempty,oneof=PENDING DEPLOYING RUNNING ERROR"`
	EnvVars      map[string]string `json:"env_vars"`
}

type deployContainerReq struct {
	RegionCode string `json:"region_code"`
}

func ListNodes(c *gin.Context) {
	region := c.Query("region_code")
	status := c.Query("status")
	minAvail := 0
	if raw := c.Query("min_available_ram_mb"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid min_available_ram_mb"})
			return
		}
		minAvail = v
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

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	rows, total, err := svc.ListNodes(region, status, minAvail, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nodes": rows, "total_count": total, "limit": limit, "offset": offset})
}

func CreateNode(c *gin.Context) {
	var req createNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	row := &models.Node{
		Name:        req.Name,
		RegionCode:  req.RegionCode,
		RegionName:  req.RegionName,
		APIEndpoint: req.APIEndpoint,
		APIKey:      req.APIKey,
		IPAddress:   req.IPAddress,
		MaxRamMB:    req.MaxRamMB,
		UsedRamMB:   req.UsedRamMB,
		Status:      req.Status,
	}
	if err := svc.CreateNode(row); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"node": row})
}

func GetNode(c *gin.Context) {
	id := c.Param("id")
	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	row, err := svc.GetNodeByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"node": row})
}

func UpdateNode(c *gin.Context) {
	id := c.Param("id")
	var req updateNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	row, err := svc.GetNodeByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	if req.Name != "" {
		row.Name = req.Name
	}
	if req.RegionCode != "" {
		row.RegionCode = req.RegionCode
	}
	if req.RegionName != "" {
		row.RegionName = req.RegionName
	}
	if req.APIEndpoint != "" {
		row.APIEndpoint = req.APIEndpoint
	}
	if req.APIKey != "" {
		row.APIKey = req.APIKey
	}
	if req.IPAddress != "" {
		row.IPAddress = req.IPAddress
	}
	if req.MaxRamMB != nil {
		row.MaxRamMB = *req.MaxRamMB
	}
	if req.UsedRamMB != nil {
		row.UsedRamMB = *req.UsedRamMB
	}
	if req.Status != "" {
		row.Status = req.Status
	}

	if err := svc.UpdateNode(row); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"node": row})
}

func DeleteNode(c *gin.Context) {
	id := c.Param("id")
	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	if err := svc.DeleteNode(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func SelectBestNode(c *gin.Context) {
	region := c.Query("region_code")
	rawRam := c.Query("required_ram_mb")
	if rawRam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "required_ram_mb is required"})
		return
	}
	reqRam, err := strconv.Atoi(rawRam)
	if err != nil || reqRam <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid required_ram_mb"})
		return
	}

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	row, err := svc.SelectBestNode(region, reqRam)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no eligible node found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"node": row})
}

func ListAppTemplates(c *gin.Context) {
	var activeOnly *bool
	if raw := c.Query("is_active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid is_active"})
			return
		}
		activeOnly = &v
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

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	rows, total, err := svc.ListAppTemplates(activeOnly, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": rows, "total_count": total, "limit": limit, "offset": offset})
}

func CreateAppTemplate(c *gin.Context) {
	var req createTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate config content if provided
	if req.ConfigContent != "" {
		typ := req.ConfigType
		if typ == "" {
			typ = "yaml"
		}
		var tmp interface{}
		switch typ {
		case "yaml":
			if err := yaml.Unmarshal([]byte(req.ConfigContent), &tmp); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid YAML format: " + err.Error()})
				return
			}
		case "json":
			if err := json.Unmarshal([]byte(req.ConfigContent), &tmp); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format: " + err.Error()})
				return
			}
		case "toml":
			if err := toml.Unmarshal([]byte(req.ConfigContent), &tmp); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid TOML format: " + err.Error()})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported config_type"})
			return
		}
	}

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	row := &models.AppTemplate{
		AppName:         req.AppName,
		DockerImage:     req.DockerImage,
		DefaultRamMB:    req.DefaultRamMB,
		DefaultCPULimit: req.DefaultCPULimit,
		ConfigContent:   req.ConfigContent,
		ConfigType:      req.ConfigType,
		IsActive:        true,
	}
	if req.IsActive != nil {
		row.IsActive = *req.IsActive
	}

	if err := svc.CreateAppTemplate(row); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"template": row})
}

func GetAppTemplate(c *gin.Context) {
	id := c.Param("id")
	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	row, err := svc.GetAppTemplateByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"template": row})
}

func UpdateAppTemplate(c *gin.Context) {
	id := c.Param("id")
	var req updateTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate config content if provided
	if req.ConfigContent != "" {
		typ := req.ConfigType
		if typ == "" {
			typ = "yaml"
		}
		var tmp interface{}
		switch typ {
		case "yaml":
			if err := yaml.Unmarshal([]byte(req.ConfigContent), &tmp); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid YAML format: " + err.Error()})
				return
			}
		case "json":
			if err := json.Unmarshal([]byte(req.ConfigContent), &tmp); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format: " + err.Error()})
				return
			}
		case "toml":
			if err := toml.Unmarshal([]byte(req.ConfigContent), &tmp); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid TOML format: " + err.Error()})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported config_type"})
			return
		}
	}

	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}

	row, err := svc.GetAppTemplateByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	if req.AppName != "" {
		row.AppName = req.AppName
	}
	if req.DockerImage != "" {
		row.DockerImage = req.DockerImage
	}
	if req.DefaultRamMB != nil {
		row.DefaultRamMB = *req.DefaultRamMB
	}
	if req.DefaultCPULimit != nil {
		row.DefaultCPULimit = *req.DefaultCPULimit
	}
	if req.ConfigContent != "" {
		row.ConfigContent = req.ConfigContent
	}
	if req.ConfigType != "" {
		row.ConfigType = req.ConfigType
	}
	if req.IsActive != nil {
		row.IsActive = *req.IsActive
	}

	if err := svc.UpdateAppTemplate(row); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"template": row})
}

func DeleteAppTemplate(c *gin.Context) {
	id := c.Param("id")
	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	if err := svc.DeleteAppTemplate(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func ListContainers(c *gin.Context) {
	customer := c.Query("customer_id")
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

	rows, total, err := svc.ListContainers(customer, nodeID, templateID, status, limit, offset)
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

func CreateContainer(c *gin.Context) {
	var req createContainerReq
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
		CustomerID:   req.CustomerID,
		NodeID:       req.NodeID,
		TemplateID:   req.TemplateID,
		ExternalID:   req.ExternalID,
		Subdomain:    req.Subdomain,
		InternalPort: req.InternalPort,
		RamMB:        req.RamMB,
		CPUPercent:   req.CPUPercent,
		Status:       req.Status,
	}

	if err := svc.CreateContainer(row, req.EnvVars); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"container": containerResponse(row)})
}

func GetContainer(c *gin.Context) {
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
	c.JSON(http.StatusOK, gin.H{"container": containerResponse(row)})
}

func UpdateContainer(c *gin.Context) {
	id := c.Param("id")
	var req updateContainerReq
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

	if req.NodeID != nil {
		row.NodeID = req.NodeID
	}
	if req.TemplateID != nil {
		row.TemplateID = req.TemplateID
	}
	if req.ExternalID != nil {
		row.ExternalID = req.ExternalID
	}
	if req.Subdomain != nil {
		row.Subdomain = req.Subdomain
	}
	if req.InternalPort != nil {
		row.InternalPort = req.InternalPort
	}
	if req.RamMB != nil {
		row.RamMB = *req.RamMB
	}
	if req.CPUPercent != nil {
		row.CPUPercent = *req.CPUPercent
	}
	if req.Status != "" {
		row.Status = req.Status
	}

	if err := svc.UpdateContainer(row, req.EnvVars); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"container": containerResponse(row)})
}

func DeleteContainer(c *gin.Context) {
	id := c.Param("id")
	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	if err := svc.DeleteContainer(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func DeployContainer(c *gin.Context) {
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

func ReconcileContainer(c *gin.Context) {
	id := c.Param("id")
	svc, err := services.NewNodeServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
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

func containerResponse(row *models.Container) gin.H {
	return gin.H{
		"id":            row.ID,
		"customer_id":   row.CustomerID,
		"node_id":       row.NodeID,
		"template_id":   row.TemplateID,
		"external_id":   row.ExternalID,
		"subdomain":     row.Subdomain,
		"internal_port": row.InternalPort,
		"ram_mb":        row.RamMB,
		"cpu_percent":   row.CPUPercent,
		"status":        row.Status,
		"env_vars":      decodeEnvVars(row.EnvVars),
		"created_at":    row.CreatedAt,
		"updated_at":    row.UpdatedAt,
	}
}

func decodeEnvVars(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	var env map[string]string
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		return nil
	}
	return env
}
