-- Replace compose with generic config columns
-- Adds `config_content` and `config_type` to `app_templates`.
ALTER TABLE app_templates ADD COLUMN IF NOT EXISTS config_content TEXT;
ALTER TABLE app_templates ADD COLUMN IF NOT EXISTS config_type VARCHAR(16) DEFAULT 'yaml';

-- Note: this migration assumes `compose_yaml` was not previously added.
-- If you had run an earlier migration that created `compose_yaml`, run the migration
-- that copies data first or adjust accordingly.
