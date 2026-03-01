-- Migration: add auth_type for node_proxies

DO $$ BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'proxy_auth_type') THEN
		CREATE TYPE proxy_auth_type AS ENUM ('user_password', 'api_key');
	END IF;
END$$;

ALTER TABLE node_proxies
	ADD COLUMN IF NOT EXISTS auth_type proxy_auth_type NOT NULL DEFAULT 'api_key';

UPDATE node_proxies
SET auth_type = CASE
	WHEN api_user IS NOT NULL AND api_password IS NOT NULL THEN 'user_password'::proxy_auth_type
	WHEN api_token IS NOT NULL THEN 'api_key'::proxy_auth_type
	ELSE 'api_key'::proxy_auth_type
END;

CREATE INDEX IF NOT EXISTS idx_node_proxies_auth_type ON node_proxies (auth_type);
