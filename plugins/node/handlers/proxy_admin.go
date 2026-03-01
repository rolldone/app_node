package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"go_framework/plugins/node/models"
	"go_framework/plugins/node/services"
)

type createProxyReq struct {
	Name        string  `json:"name" binding:"required"`
	ProxyType   string  `json:"proxy_type" binding:"required,oneof=caddy-manager npm"`
	AuthType    string  `json:"auth_type" binding:"required,oneof=user_password api_key"`
	APIURL      string  `json:"api_url" binding:"required,url"`
	APIUser     *string `json:"api_user" binding:"omitempty"`
	APIPassword *string `json:"api_password" binding:"omitempty"`
	APIToken    *string `json:"api_token" binding:"omitempty"`
	IsActive    *bool   `json:"is_active" binding:"omitempty"`
}

type updateProxyReq struct {
	Name        *string `json:"name" binding:"omitempty"`
	AuthType    *string `json:"auth_type" binding:"omitempty,oneof=user_password api_key"`
	APIURL      *string `json:"api_url" binding:"omitempty,url"`
	APIUser     *string `json:"api_user" binding:"omitempty"`
	APIPassword *string `json:"api_password" binding:"omitempty"`
	APIToken    *string `json:"api_token" binding:"omitempty"`
	IsActive    *bool   `json:"is_active" binding:"omitempty"`
}

type assignProxyReq struct {
	ProxyID *string `json:"proxy_id" binding:"omitempty,uuid"`
}

func ListProxies(c *gin.Context) {
	svc, err := services.NewNodeProxyServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	active := c.Query("active")
	activeOnly := false
	if active != "" {
		if active == "true" || active == "1" {
			activeOnly = true
		}
	}
	rows, err := svc.List(activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"proxies": rows})
}

func GetProxy(c *gin.Context) {
	id := c.Param("id")
	svc, err := services.NewNodeProxyServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	p, err := svc.Get(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "proxy not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"proxy": p})
}

func CreateProxy(c *gin.Context) {
	var req createProxyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	psvc, err := services.NewNodeProxyServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	p := &models.NodeProxy{
		Name:      req.Name,
		ProxyType: req.ProxyType,
		AuthType:  req.AuthType,
		APIURL:    req.APIURL,
	}
	if req.APIUser != nil {
		p.APIUser = req.APIUser
	}
	if req.APIPassword != nil {
		p.APIPassword = req.APIPassword
	}
	if req.APIToken != nil {
		p.APIToken = req.APIToken
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	} else {
		p.IsActive = true
	}
	if err := psvc.Create(p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"proxy": p})
}

func UpdateProxy(c *gin.Context) {
	id := c.Param("id")
	var req updateProxyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	psvc, err := services.NewNodeProxyServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	changes := make(map[string]interface{})
	if req.Name != nil {
		changes["name"] = *req.Name
	}
	if req.AuthType != nil {
		changes["auth_type"] = *req.AuthType
	}
	if req.APIURL != nil {
		changes["api_url"] = *req.APIURL
	}
	if req.APIUser != nil {
		if strings.TrimSpace(*req.APIUser) != "" {
			changes["api_user"] = *req.APIUser
		}
	}
	if req.APIPassword != nil {
		if strings.TrimSpace(*req.APIPassword) != "" {
			changes["api_password"] = *req.APIPassword
		}
	}
	if req.APIToken != nil {
		if strings.TrimSpace(*req.APIToken) != "" {
			changes["api_token"] = *req.APIToken
		}
	}
	if req.IsActive != nil {
		changes["is_active"] = *req.IsActive
	}
	p, err := psvc.Update(id, changes)
	if err != nil {
		if errors.Is(err, services.ErrInvalidProxyAuthType) || errors.Is(err, services.ErrInvalidProxyCredential) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"proxy": p})
}

func DeleteProxy(c *gin.Context) {
	id := c.Param("id")
	psvc, err := services.NewNodeProxyServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	if err := psvc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func ToggleProxy(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	psvc, err := services.NewNodeProxyServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	if err := psvc.ToggleActive(id, body.Active); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func AssignProxyToNode(c *gin.Context) {
	nodeID := c.Param("id")
	var req assignProxyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	psvc, err := services.NewNodeProxyServiceFromDefault()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "db unavailable"})
		return
	}
	// use NodeProxyService.AssignProxy to update node.proxy_id
	if err := psvc.AssignProxy(nodeID, req.ProxyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
