DROP TABLE IF EXISTS cdr_processed_calls;
DROP TABLE IF EXISTS daily_cdr_summary;
ALTER TABLE calls DROP INDEX idx_calls_customer;
ALTER TABLE calls DROP COLUMN customer_id;
