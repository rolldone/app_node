-- Rollback: remove auth_type for node_proxies

DROP INDEX IF EXISTS idx_node_proxies_auth_type;

ALTER TABLE node_proxies
	DROP COLUMN IF EXISTS auth_type;

DROP TYPE IF EXISTS proxy_auth_type;
