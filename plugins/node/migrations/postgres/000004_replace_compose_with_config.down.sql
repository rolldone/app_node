-- Rollback: remove generic config columns
ALTER TABLE app_templates DROP COLUMN IF EXISTS config_content;
ALTER TABLE app_templates DROP COLUMN IF EXISTS config_type;
