-- Enums
DO $$ BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'node_status') THEN
		CREATE TYPE node_status AS ENUM ('ACTIVE', 'MAINTENANCE', 'FULL');
	END IF;
END$$;

DO $$ BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'container_status') THEN
		CREATE TYPE container_status AS ENUM ('PENDING', 'DEPLOYING', 'RUNNING', 'ERROR');
	END IF;
END$$;

-- Table: nodes
CREATE TABLE IF NOT EXISTS nodes (
	id UUID PRIMARY KEY,
	name VARCHAR(100) NOT NULL,
	region_code VARCHAR(10) NOT NULL,
	region_name VARCHAR(50) NOT NULL,
	api_endpoint TEXT NOT NULL,
	api_key TEXT NOT NULL,
	ip_address INET NOT NULL,
	max_ram_mb INTEGER NOT NULL CHECK (max_ram_mb > 0),
	used_ram_mb INTEGER NOT NULL DEFAULT 0 CHECK (used_ram_mb >= 0 AND used_ram_mb <= max_ram_mb),
	status node_status NOT NULL DEFAULT 'ACTIVE',
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Table: app_templates
CREATE TABLE IF NOT EXISTS app_templates (
	id UUID PRIMARY KEY,
	app_name VARCHAR(50) NOT NULL,
	docker_image TEXT NOT NULL,
	default_ram_mb INTEGER NOT NULL DEFAULT 512 CHECK (default_ram_mb > 0),
	default_cpu_limit INTEGER NOT NULL DEFAULT 50 CHECK (default_cpu_limit > 0),
	config_schema JSONB,
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Table: containers
CREATE TABLE IF NOT EXISTS containers (
	id UUID PRIMARY KEY,
	customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
	node_id UUID REFERENCES nodes(id),
	template_id UUID REFERENCES app_templates(id),
	external_id VARCHAR(100),
	subdomain VARCHAR(255) UNIQUE,
	internal_port INTEGER CHECK (internal_port IS NULL OR (internal_port >= 1 AND internal_port <= 65535)),
	ram_mb INTEGER NOT NULL CHECK (ram_mb > 0),
	cpu_percent INTEGER NOT NULL CHECK (cpu_percent > 0),
	status container_status NOT NULL DEFAULT 'PENDING',
	env_vars JSONB,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_nodes_region ON nodes(region_code);
CREATE INDEX IF NOT EXISTS idx_containers_customer ON containers(customer_id);
CREATE INDEX IF NOT EXISTS idx_containers_node ON containers(node_id);
CREATE INDEX IF NOT EXISTS idx_containers_status ON containers(status);
