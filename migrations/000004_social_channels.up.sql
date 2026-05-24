-- Social Channels: WeChat / Weibo platform config
CREATE TABLE IF NOT EXISTS social_channel_configs (
    id BIGINT NOT NULL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    platform VARCHAR(20) NOT NULL,
    app_id VARCHAR(255) NOT NULL,
    app_secret VARCHAR(512) NOT NULL,
    token VARCHAR(255) NOT NULL,
    encoding_aes_key VARCHAR(512) NOT NULL DEFAULT '',
    webhook_url VARCHAR(1024) NOT NULL DEFAULT '',
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME(3) NOT NULL,
    updated_at DATETIME(3) NOT NULL,
    UNIQUE INDEX idx_scc_channel (channel_id),
    INDEX idx_scc_tenant (tenant_id),
    INDEX idx_scc_platform_app (platform, app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
