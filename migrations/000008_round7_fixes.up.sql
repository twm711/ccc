-- Round 7: Add lock_expires_at to ivr_flows for lock TTL support (P1-9).
ALTER TABLE ivr_flows ADD COLUMN IF NOT EXISTS lock_expires_at DATETIME NULL AFTER locked_at;

-- Round 7: Add hangup_by to calls if not already present (P1-4).
-- The column already exists in 000001_init_schema.up.sql for fresh installs.
-- This handles upgrades from older schemas.
ALTER TABLE calls ADD COLUMN IF NOT EXISTS hangup_by ENUM('agent','customer','system') NULL AFTER hangup_reason;
