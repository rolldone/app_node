DROP TABLE IF EXISTS containers;
DROP TABLE IF EXISTS app_templates;
DROP TABLE IF EXISTS nodes;

DO $$ BEGIN
	IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'container_status') THEN
		DROP TYPE container_status;
	END IF;
END$$;

DO $$ BEGIN
	IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'node_status') THEN
		DROP TYPE node_status;
	END IF;
END$$;
