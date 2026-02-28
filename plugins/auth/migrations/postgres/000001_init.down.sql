-- Drop tables in reverse order
DROP TABLE IF EXISTS admin_password_resets;
DROP TABLE IF EXISTS admin_audit_logs;
DROP TABLE IF EXISTS admin_api_keys;
DROP TABLE IF EXISTS admin_sessions;
DROP TABLE IF EXISTS admins;

-- Drop enum type
DO $$ BEGIN
	IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'admin_level') THEN
		DROP TYPE admin_level;
	END IF;
END$$;
