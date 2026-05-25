-- Round 10 — bridge calls<->CRM, add encryption metadata writes, daily CDR aggregation.

-- SILO-R10-1: add calls.customer_id so CRM linkage actually persists.
ALTER TABLE calls
  ADD COLUMN IF NOT EXISTS customer_id BIGINT UNSIGNED NULL AFTER campaign_case_id;
ALTER TABLE calls
  ADD INDEX idx_calls_customer (customer_id, started_at);

-- OPS-R10-5: daily CDR aggregation table for billing/operational reporting.
CREATE TABLE IF NOT EXISTS daily_cdr_summary (
  id               BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  tenant_id        BIGINT UNSIGNED NOT NULL,
  bucket_date      DATE NOT NULL,
  inbound_calls    INT UNSIGNED NOT NULL DEFAULT 0,
  outbound_calls   INT UNSIGNED NOT NULL DEFAULT 0,
  answered_calls   INT UNSIGNED NOT NULL DEFAULT 0,
  abandoned_calls  INT UNSIGNED NOT NULL DEFAULT 0,
  total_talk_sec   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  total_ring_sec   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  total_queue_sec  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  generated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_tenant_date (tenant_id, bucket_date),
  INDEX idx_date (bucket_date)
);
