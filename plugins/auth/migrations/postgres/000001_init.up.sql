-- Enums
DO $$ BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'admin_level') THEN
		CREATE TYPE admin_level AS ENUM ('STAFF', 'SUPERADMIN');
	END IF;
END$$;

-- Table: admins
CREATE TABLE IF NOT EXISTS admins (
	id UUID PRIMARY KEY,
	username VARCHAR(100) UNIQUE NOT NULL,
	email VARCHAR(255) UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	level admin_level DEFAULT 'STAFF',
	is_active BOOLEAN DEFAULT true,
	last_login_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Table: admin_sessions
CREATE TABLE IF NOT EXISTS admin_sessions (
	id UUID PRIMARY KEY,
	admin_id UUID REFERENCES admins(id) ON DELETE CASCADE,
	refresh_token_hash TEXT NOT NULL,
	user_agent TEXT,
	ip_address INET,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	expires_at TIMESTAMPTZ,
	revoked BOOLEAN DEFAULT false
);

-- Table: admin_api_keys
CREATE TABLE IF NOT EXISTS admin_api_keys (
	id UUID PRIMARY KEY,
	admin_id UUID REFERENCES admins(id) ON DELETE SET NULL,
	name VARCHAR(255),
	key_hash TEXT NOT NULL,
	scopes JSONB,
	is_active BOOLEAN DEFAULT true,
	created_at TIMESTAMPTZ DEFAULT NOW(),
	revoked_at TIMESTAMPTZ
);

-- Table: admin_audit_logs
CREATE TABLE IF NOT EXISTS admin_audit_logs (
	id UUID PRIMARY KEY,
	admin_id UUID,
	action VARCHAR(100) NOT NULL,
	target_type VARCHAR(100),
	target_id UUID,
	meta JSONB,
	created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Table: admin_password_resets
CREATE TABLE IF NOT EXISTS admin_password_resets (
	id UUID PRIMARY KEY,
	admin_id UUID REFERENCES admins(id) ON DELETE CASCADE,
	token_hash TEXT NOT NULL,
	expires_at TIMESTAMPTZ,
	used BOOLEAN DEFAULT false,
	created_at TIMESTAMPTZ DEFAULT NOW()
);
