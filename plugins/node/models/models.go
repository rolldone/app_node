package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	appuuid "go_framework/internal/uuid"

	"gorm.io/gorm"
)

// NodeMetadata represents additional node configuration stored as JSON
type NodeMetadata map[string]interface{}

// Scan implements sql.Scanner interface for JSONB
func (m *NodeMetadata) Scan(value interface{}) error {
	if value == nil {
		*m = make(NodeMetadata)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// Value implements driver.Valuer interface for JSONB
func (m NodeMetadata) Value() (driver.Value, error) {
	if m == nil {
		return json.Marshal(make(NodeMetadata))
	}
	return json.Marshal(m)
}

// NodeProxy represents a proxy manager (CaddyManager or NPM)
type NodeProxy struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"size:100;not null;uniqueIndex" json:"name"`
	ProxyType   string    `gorm:"type:proxy_type;not null" json:"proxy_type"`
	AuthType    string    `gorm:"type:proxy_auth_type;not null;default:api_key" json:"auth_type"`
	APIURL      string    `gorm:"type:text;not null" json:"api_url"`
	APIUser     *string   `gorm:"size:100" json:"api_user,omitempty"`
	APIPassword *string   `gorm:"type:text;column:api_password" json:"api_password,omitempty"`
	APIToken    *string   `gorm:"type:text" json:"api_token,omitempty"`
	IsActive    bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (NodeProxy) TableName() string { return "node_proxies" }

func (p *NodeProxy) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		id, err := appuuid.New()
		if err != nil {
			return err
		}
		p.ID = id
	}
	return nil
}

type Node struct {
	ID          string       `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string       `gorm:"size:100;not null" json:"name"`
	RegionCode  string       `gorm:"size:10;not null" json:"region_code"`
	RegionName  string       `gorm:"size:50;not null" json:"region_name"`
	APIEndpoint string       `gorm:"type:text;not null" json:"api_endpoint"`
	APIKey      string       `gorm:"type:text;not null" json:"api_key"`
	IPAddress   string       `gorm:"type:inet;not null" json:"ip_address"`
	MaxRamMB    int          `gorm:"not null" json:"max_ram_mb"`
	UsedRamMB   int          `gorm:"not null;default:0" json:"used_ram_mb"`
	Status      string       `gorm:"type:node_status;not null;default:ACTIVE" json:"status"`
	ProxyID     *string      `gorm:"type:uuid" json:"proxy_id,omitempty"`
	Metadata    NodeMetadata `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`

	// Relations
	Proxy *NodeProxy `gorm:"foreignKey:ProxyID" json:"proxy,omitempty"`
}

func (Node) TableName() string { return "nodes" }

func (n *Node) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		id, err := appuuid.New()
		if err != nil {
			return err
		}
		n.ID = id
	}
	return nil
}

type AppTemplate struct {
	ID              string    `gorm:"type:uuid;primaryKey" json:"id"`
	AppName         string    `gorm:"size:50;not null" json:"app_name"`
	DockerImage     string    `gorm:"type:text;not null" json:"docker_image"`
	DefaultRamMB    int       `gorm:"not null;default:512" json:"default_ram_mb"`
	DefaultCPULimit int       `gorm:"not null;default:50" json:"default_cpu_limit"`
	ConfigContent   string    `gorm:"type:text" json:"config_content"`
	ConfigType      string    `gorm:"size:16" json:"config_type"`
	IsActive        bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (AppTemplate) TableName() string { return "app_templates" }

func (t *AppTemplate) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		id, err := appuuid.New()
		if err != nil {
			return err
		}
		t.ID = id
	}
	return nil
}

type Container struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	CustomerID   string    `gorm:"type:uuid;not null" json:"customer_id"`
	NodeID       *string   `gorm:"type:uuid" json:"node_id,omitempty"`
	TemplateID   *string   `gorm:"type:uuid" json:"template_id,omitempty"`
	ExternalID   *string   `gorm:"size:100" json:"external_id,omitempty"`
	Subdomain    *string   `gorm:"size:255;uniqueIndex" json:"subdomain,omitempty"`
	InternalPort *int      `json:"internal_port,omitempty"`
	RamMB        int       `gorm:"not null" json:"ram_mb"`
	CPUPercent   int       `gorm:"not null" json:"cpu_percent"`
	Status       string    `gorm:"type:container_status;not null;default:PENDING" json:"status"`
	EnvVars      string    `gorm:"type:jsonb" json:"env_vars"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Container) TableName() string { return "containers" }

func (c *Container) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		id, err := appuuid.New()
		if err != nil {
			return err
		}
		c.ID = id
	}
	return nil
}
