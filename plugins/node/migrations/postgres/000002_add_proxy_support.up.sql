-- Migration: Add proxy support for nodes
-- Description: Creates node_proxies table and adds proxy relationship to nodes table

-- Create enum for proxy types
DO $$ BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'proxy_type') THEN
		CREATE TYPE proxy_type AS ENUM ('caddy-manager', 'npm');
	END IF;
END$$;

-- Create node_proxies table
CREATE TABLE IF NOT EXISTS node_proxies (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    proxy_type proxy_type NOT NULL,
    api_url TEXT NOT NULL,
    api_user VARCHAR(100),
    api_password TEXT,
    api_token TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_node_proxies_type ON node_proxies (proxy_type);
CREATE INDEX IF NOT EXISTS idx_node_proxies_active ON node_proxies (is_active);

-- Alter nodes table to add proxy_id and metadata
-- proxy_id is nullable (nodes can exist without proxy)
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS proxy_id UUID REFERENCES node_proxies(id) ON DELETE SET NULL;
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;

-- Add index on proxy_id for join performance
CREATE INDEX IF NOT EXISTS idx_nodes_proxy_id ON nodes (proxy_id);

-- Add index on metadata for JSONB queries
CREATE INDEX IF NOT EXISTS idx_nodes_metadata ON nodes USING GIN (metadata);
