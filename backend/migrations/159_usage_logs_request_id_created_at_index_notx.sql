CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_request_id_created_at
    ON usage_logs (request_id, created_at DESC);
