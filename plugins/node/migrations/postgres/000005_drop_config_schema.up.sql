-- 000005_drop_config_schema.up.sql
-- Minimal: drop legacy `config_schema` column from the correct table `app_templates`.
BEGIN;

ALTER TABLE app_templates
  DROP COLUMN IF EXISTS config_schema;

COMMIT;
