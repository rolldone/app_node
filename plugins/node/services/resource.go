package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go_framework/internal/db"
	"go_framework/plugins/node/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrContainerNotFound = errors.New("container not found")
	ErrTemplateNotFound  = errors.New("template not found")
	ErrTemplateInactive  = errors.New("template is inactive")
	ErrNodeNotFound      = errors.New("node not found")
	ErrNoEligibleNode    = errors.New("no eligible node found")
	ErrInvalidState      = errors.New("invalid container state for deploy")
	ErrDeployRequest     = errors.New("deploy request failed")
)

type NodeService struct {
	db *gorm.DB
}

func NewNodeService(gdb *gorm.DB) (*NodeService, error) {
	if gdb == nil {
		return nil, errors.New("db is nil")
	}
	return &NodeService{db: gdb}, nil
}

func NewNodeServiceFromDefault() (*NodeService, error) {
	gdb, err := db.GetGormDB()
	if err != nil {
		return nil, err
	}
	return NewNodeService(gdb)
}

func (s *NodeService) ListNodes(regionCode string, status string, minAvailableRamMB int, limit int, offset int) ([]models.Node, int64, error) {
	query := s.db.Model(&models.Node{}).Preload("Proxy")
	if regionCode != "" {
		query = query.Where("region_code = ?", regionCode)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if minAvailableRamMB > 0 {
		query = query.Where("(max_ram_mb - used_ram_mb) >= ?", minAvailableRamMB)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var rows []models.Node
	if err := query.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *NodeService) GetNodeByID(id string) (*models.Node, error) {
	var row models.Node
	if err := s.db.Preload("Proxy").Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *NodeService) CreateNode(in *models.Node) error {
	if in.Status == "" {
		in.Status = "ACTIVE"
	}
	return s.db.Create(in).Error
}

func (s *NodeService) UpdateNode(in *models.Node) error {
	return s.db.Save(in).Error
}

func (s *NodeService) DeleteNode(id string) error {
	return s.db.Delete(&models.Node{}, "id = ?", id).Error
}

func (s *NodeService) SelectBestNode(regionCode string, requiredRamMB int) (*models.Node, error) {
	if requiredRamMB <= 0 {
		return nil, errors.New("required_ram_mb must be greater than zero")
	}

	query := s.db.Model(&models.Node{}).Where("status = ?", "ACTIVE").Where("(max_ram_mb - used_ram_mb) >= ?", requiredRamMB)
	if regionCode != "" {
		query = query.Where("region_code = ?", regionCode)
	}

	var row models.Node
	if err := query.Preload("Proxy").Order("(max_ram_mb - used_ram_mb) DESC, created_at ASC").First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// GetNodeMetadata returns the metadata JSONB for a node.
func (s *NodeService) GetNodeMetadata(id string) (models.NodeMetadata, error) {
	var row models.Node
	if err := s.db.Select("metadata").Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return row.Metadata, nil
}

// UpdateNodeMetadata replaces the metadata for a node.
func (s *NodeService) UpdateNodeMetadata(id string, metadata models.NodeMetadata) error {
	return s.db.Model(&models.Node{}).Where("id = ?", id).Updates(map[string]interface{}{"metadata": metadata, "updated_at": time.Now().UTC()}).Error
}

// AssignProxy assigns or unassigns a proxy to a node. proxyID nil will unassign.
func (s *NodeService) AssignProxy(nodeID string, proxyID *string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if proxyID != nil {
			// validate proxy exists and active
			var p models.NodeProxy
			if err := tx.Where("id = ? AND is_active = ?", *proxyID, true).First(&p).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrNodeNotFound
				}
				return err
			}
		}

		// prepare value for update
		var val interface{}
		if proxyID == nil {
			val = nil
		} else {
			val = *proxyID
		}

		if err := tx.Model(&models.Node{}).Where("id = ?", nodeID).Updates(map[string]interface{}{"proxy_id": val, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *NodeService) ListAppTemplates(activeOnly *bool, limit int, offset int) ([]models.AppTemplate, int64, error) {
	query := s.db.Model(&models.AppTemplate{})
	if activeOnly != nil {
		query = query.Where("is_active = ?", *activeOnly)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var rows []models.AppTemplate
	if err := query.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *NodeService) GetAppTemplateByID(id string) (*models.AppTemplate, error) {
	var row models.AppTemplate
	if err := s.db.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *NodeService) CreateAppTemplate(in *models.AppTemplate) error {
	if in.DefaultRamMB <= 0 {
		in.DefaultRamMB = 512
	}
	if in.DefaultCPULimit <= 0 {
		in.DefaultCPULimit = 50
	}

	return s.db.Create(in).Error
}

func (s *NodeService) UpdateAppTemplate(in *models.AppTemplate) error {
	return s.db.Save(in).Error
}

func (s *NodeService) DeleteAppTemplate(id string) error {
	return s.db.Delete(&models.AppTemplate{}, "id = ?", id).Error
}

func (s *NodeService) ListContainers(customerID, nodeID, templateID, status string, limit int, offset int) ([]models.Container, int64, error) {
	query := s.db.Model(&models.Container{})
	if customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}
	if nodeID != "" {
		query = query.Where("node_id = ?", nodeID)
	}
	if templateID != "" {
		query = query.Where("template_id = ?", templateID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var rows []models.Container
	if err := query.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *NodeService) GetContainerByID(id string) (*models.Container, error) {
	var row models.Container
	if err := s.db.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *NodeService) CreateContainer(in *models.Container, envVars map[string]string) error {
	if in.CustomerID == "" {
		return errors.New("customer_id is required")
	}
	if in.RamMB <= 0 {
		return errors.New("ram_mb must be greater than zero")
	}
	if in.CPUPercent <= 0 {
		return errors.New("cpu_percent must be greater than zero")
	}
	if in.Status == "" {
		in.Status = "PENDING"
	}
	if envVars != nil {
		b, err := json.Marshal(envVars)
		if err != nil {
			return err
		}
		in.EnvVars = string(b)
	}
	return s.db.Create(in).Error
}

func (s *NodeService) UpdateContainer(in *models.Container, envVars map[string]string) error {
	if envVars != nil {
		b, err := json.Marshal(envVars)
		if err != nil {
			return err
		}
		in.EnvVars = string(b)
	}
	return s.db.Save(in).Error
}

func (s *NodeService) DeleteContainer(id string) error {
	var container models.Container
	if err := s.db.Where("id = ?", id).First(&container).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if err := s.db.Delete(&models.Container{}, "id = ?", id).Error; err != nil {
		return err
	}

	if container.NodeID != nil && *container.NodeID != "" && container.Status == "RUNNING" {
		_ = s.releaseNodeResource(*container.NodeID, container.RamMB)
	}

	return nil
}

func (s *NodeService) DeployContainer(id string, regionCode string) (*models.Container, error) {
	container, node, template, err := s.prepareDeploy(id, regionCode)
	if err != nil {
		return nil, err
	}

	externalID, internalPort, err := s.callNodeDeploy(node, container, template)
	if err != nil {
		_ = s.failDeploy(container.ID, node.ID, container.RamMB)
		return nil, err
	}

	if err := s.finalizeDeploy(container.ID, externalID, internalPort); err != nil {
		return nil, err
	}

	return s.GetContainerByID(container.ID)
}

func (s *NodeService) prepareDeploy(containerID, regionCode string) (*models.Container, *models.Node, *models.AppTemplate, error) {
	var selectedContainer models.Container
	var selectedNode models.Node
	var selectedTemplate models.AppTemplate

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", containerID).First(&selectedContainer).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrContainerNotFound
			}
			return err
		}

		if selectedContainer.Status == "DEPLOYING" || selectedContainer.Status == "RUNNING" {
			return fmt.Errorf("%w: current status %s", ErrInvalidState, selectedContainer.Status)
		}

		if selectedContainer.TemplateID == nil || *selectedContainer.TemplateID == "" {
			return ErrTemplateNotFound
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *selectedContainer.TemplateID).First(&selectedTemplate).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTemplateNotFound
			}
			return err
		}
		if !selectedTemplate.IsActive {
			return ErrTemplateInactive
		}

		node, err := s.resolveDeployNode(tx, &selectedContainer, regionCode)
		if err != nil {
			return err
		}
		selectedNode = *node

		selectedContainer.NodeID = &selectedNode.ID
		selectedContainer.Status = "DEPLOYING"
		if err := tx.Save(&selectedContainer).Error; err != nil {
			return err
		}

		selectedNode.UsedRamMB += selectedContainer.RamMB
		if selectedNode.UsedRamMB >= selectedNode.MaxRamMB {
			selectedNode.Status = "FULL"
		}
		if err := tx.Save(&selectedNode).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return &selectedContainer, &selectedNode, &selectedTemplate, nil
}

func (s *NodeService) resolveDeployNode(tx *gorm.DB, container *models.Container, regionCode string) (*models.Node, error) {
	var node models.Node

	if container.NodeID != nil && *container.NodeID != "" {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", *container.NodeID).First(&node).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrNodeNotFound
			}
			return nil, err
		}
		if node.Status != "ACTIVE" {
			return nil, ErrNoEligibleNode
		}
		if node.MaxRamMB-node.UsedRamMB < container.RamMB {
			return nil, ErrNoEligibleNode
		}
		return &node, nil
	}

	query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&models.Node{}).Where("status = ?", "ACTIVE").Where("(max_ram_mb - used_ram_mb) >= ?", container.RamMB)
	if regionCode != "" {
		query = query.Where("region_code = ?", regionCode)
	}
	if err := query.Order("(max_ram_mb - used_ram_mb) DESC, created_at ASC").First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoEligibleNode
		}
		return nil, err
	}
	return &node, nil
}

func (s *NodeService) callNodeDeploy(node *models.Node, container *models.Container, template *models.AppTemplate) (string, *int, error) {
	type deployRequest struct {
		ContainerID string            `json:"container_id"`
		CustomerID  string            `json:"customer_id"`
		TemplateID  string            `json:"template_id"`
		AppName     string            `json:"app_name"`
		DockerImage string            `json:"docker_image"`
		RamMB       int               `json:"ram_mb"`
		CPUPercent  int               `json:"cpu_percent"`
		EnvVars     map[string]string `json:"env_vars,omitempty"`
	}

	type deployResponse struct {
		ExternalID   string `json:"external_id"`
		ID           string `json:"id"`
		InternalPort *int   `json:"internal_port"`
		Port         *int   `json:"port"`
	}

	endpoint := strings.TrimRight(node.APIEndpoint, "/") + "/deploy"
	payload := deployRequest{
		ContainerID: container.ID,
		CustomerID:  container.CustomerID,
		TemplateID:  deref(container.TemplateID),
		AppName:     template.AppName,
		DockerImage: template.DockerImage,
		RamMB:       container.RamMB,
		CPUPercent:  container.CPUPercent,
		EnvVars:     decodeEnvVarsMap(container.EnvVars),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+node.APIKey)
	req.Header.Set("X-API-Key", node.APIKey)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrDeployRequest, err)
	}
	defer resp.Body.Close()

	rawResp, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(rawResp))
		if msg == "" {
			msg = resp.Status
		}
		return "", nil, fmt.Errorf("%w: %s", ErrDeployRequest, msg)
	}

	out := deployResponse{}
	if len(rawResp) > 0 {
		if err := json.Unmarshal(rawResp, &out); err != nil {
			return "", nil, fmt.Errorf("%w: invalid response body", ErrDeployRequest)
		}
	}

	externalID := out.ExternalID
	if externalID == "" {
		externalID = out.ID
	}
	if externalID == "" {
		externalID = fmt.Sprintf("node-%s", container.ID)
	}

	internalPort := out.InternalPort
	if internalPort == nil {
		internalPort = out.Port
	}

	return externalID, internalPort, nil
}

func (s *NodeService) finalizeDeploy(containerID, externalID string, internalPort *int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var row models.Container
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", containerID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrContainerNotFound
			}
			return err
		}

		row.Status = "RUNNING"
		if externalID != "" {
			row.ExternalID = &externalID
		}
		if internalPort != nil {
			row.InternalPort = internalPort
		}
		return tx.Save(&row).Error
	})
}

func (s *NodeService) failDeploy(containerID, nodeID string, ramMB int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Container{}).Where("id = ?", containerID).Updates(map[string]any{
			"status": "ERROR",
		}).Error; err != nil {
			return err
		}

		var node models.Node
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", nodeID).First(&node).Error; err != nil {
			return err
		}

		node.UsedRamMB -= ramMB
		if node.UsedRamMB < 0 {
			node.UsedRamMB = 0
		}
		if node.Status == "FULL" && node.UsedRamMB < node.MaxRamMB {
			node.Status = "ACTIVE"
		}

		return tx.Save(&node).Error
	})
}

func deref(in *string) string {
	if in == nil {
		return ""
	}
	return *in
}

func decodeEnvVarsMap(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	var env map[string]string
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		return nil
	}
	return env
}

func (s *NodeService) releaseNodeResource(nodeID string, ramMB int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var node models.Node
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", nodeID).First(&node).Error; err != nil {
			return err
		}

		node.UsedRamMB -= ramMB
		if node.UsedRamMB < 0 {
			node.UsedRamMB = 0
		}
		if node.Status == "FULL" && node.UsedRamMB < node.MaxRamMB {
			node.Status = "ACTIVE"
		}

		return tx.Save(&node).Error
	})
}

func (s *NodeService) ReconcileContainer(id string) (*models.Container, error) {
	var container models.Container
	if err := s.db.Where("id = ?", id).First(&container).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrContainerNotFound
		}
		return nil, err
	}

	if container.NodeID == nil || *container.NodeID == "" {
		return &container, nil
	}
	if container.ExternalID == nil || *container.ExternalID == "" {
		return &container, nil
	}

	var node models.Node
	if err := s.db.Where("id = ?", *container.NodeID).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &container, nil
		}
		return nil, err
	}

	nodeStatus, err := s.queryNodeContainerStatus(&node, *container.ExternalID)
	if err != nil {
		return &container, nil
	}

	if nodeStatus != "" && nodeStatus != container.Status {
		container.Status = nodeStatus
		if err := s.db.Save(&container).Error; err != nil {
			return nil, err
		}
	}

	return &container, nil
}

func (s *NodeService) queryNodeContainerStatus(node *models.Node, externalID string) (string, error) {
	type statusResponse struct {
		Status string `json:"status"`
		State  string `json:"state"`
	}

	endpoint := strings.TrimRight(node.APIEndpoint, "/") + "/containers/" + externalID + "/status"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+node.APIKey)
	req.Header.Set("X-API-Key", node.APIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "ERROR", nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("node API status %d", resp.StatusCode)
	}

	rawResp, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if len(rawResp) == 0 {
		return "", nil
	}

	out := statusResponse{}
	if err := json.Unmarshal(rawResp, &out); err != nil {
		return "", err
	}

	if out.Status != "" {
		return strings.ToUpper(out.Status), nil
	}
	if out.State != "" {
		return strings.ToUpper(out.State), nil
	}

	return "", nil
}
