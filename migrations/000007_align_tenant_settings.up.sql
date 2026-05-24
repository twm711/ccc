-- Round 6 P1-4: align tenant_settings with the Go entity expectations.
--
-- The entity (identity.TenantSettings) carries MaxAgents and
-- RecordingStorageBackend fields that have no DB column today, while the
-- existing MySQL repo aliases unrelated columns (`default_acw_seconds AS
-- max_agents`) which corrupts the values on every read/write. Add the
-- missing columns and seed sane defaults so reads round-trip correctly.

ALTER TABLE tenant_settings
  ADD COLUMN max_agents INT UNSIGNED NOT NULL DEFAULT 50
    COMMENT '租户最大坐席数' AFTER tenant_id,
  ADD COLUMN recording_storage_backend VARCHAR(32) NOT NULL DEFAULT 'local'
    COMMENT '录音存储后端 local|minio|s3' AFTER recording_retention_days;
