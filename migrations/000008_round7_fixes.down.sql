ALTER TABLE ivr_flows DROP COLUMN IF EXISTS lock_expires_at;
-- hangup_by is kept as it exists in the initial schema.
