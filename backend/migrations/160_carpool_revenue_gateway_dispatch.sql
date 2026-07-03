ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_dispatch_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_shadow_mode BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_traffic_percent DECIMAL(8,4) NOT NULL DEFAULT 0;
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_allowed_group_ids TEXT NOT NULL DEFAULT '';
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_allowed_models TEXT NOT NULL DEFAULT '';
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_min_remain_ratio DECIMAL(8,4) NOT NULL DEFAULT 0.1000;
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_max_daily_quota DECIMAL(20,8) NOT NULL DEFAULT 0;

ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS subscription_id BIGINT NULL REFERENCES user_subscriptions(id) ON DELETE SET NULL;
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS request_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS request_api_key_id BIGINT NULL REFERENCES api_keys(id) ON DELETE SET NULL;
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS request_account_id BIGINT NULL REFERENCES accounts(id) ON DELETE SET NULL;
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS request_group_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL;
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS dispatch_mode VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS decision_reason TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_carpool_revenue_records_request_user ON carpool_revenue_records (request_user_id);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_records_request_api_key ON carpool_revenue_records (request_api_key_id);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_records_request_account ON carpool_revenue_records (request_account_id);
