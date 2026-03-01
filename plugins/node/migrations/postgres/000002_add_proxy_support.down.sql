-- Rollback: Remove proxy support from nodes

-- Drop indexes
DROP INDEX IF EXISTS idx_nodes_proxy_id;
DROP INDEX IF EXISTS idx_nodes_metadata;
DROP INDEX IF EXISTS idx_node_proxies_type;
DROP INDEX IF EXISTS idx_node_proxies_active;

-- Remove columns from nodes table
ALTER TABLE nodes DROP COLUMN IF EXISTS proxy_id;
ALTER TABLE nodes DROP COLUMN IF EXISTS metadata;

-- Drop node_proxies table
DROP TABLE IF EXISTS node_proxies;

-- Drop proxy_type enum
DROP TYPE IF EXISTS proxy_type;
