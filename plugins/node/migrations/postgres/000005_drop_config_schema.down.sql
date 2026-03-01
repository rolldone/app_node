-- 000005_drop_config_schema.down.sql
-- Recreate `config_schema` column (type JSONB) on `app_templates`.
BEGIN;

ALTER TABLE app_templates
  ADD COLUMN IF NOT EXISTS config_schema JSONB;

COMMIT;
