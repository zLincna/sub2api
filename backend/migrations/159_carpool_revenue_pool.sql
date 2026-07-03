CREATE TABLE IF NOT EXISTS carpool_revenue_configs (
    id BIGINT PRIMARY KEY DEFAULT 1,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    user_share_ratio DECIMAL(8,4) NOT NULL DEFAULT 0.7000,
    platform_share_ratio DECIMAL(8,4) NOT NULL DEFAULT 0.3000,
    min_withdraw_amount DECIMAL(20,8) NOT NULL DEFAULT 1.00000000,
    withdraw_cooldown_minutes INTEGER NOT NULL DEFAULT 0,
    settlement_cycle VARCHAR(32) NOT NULL DEFAULT 'manual',
    freeze_minutes INTEGER NOT NULL DEFAULT 0,
    allow_user_withdraw BOOLEAN NOT NULL DEFAULT TRUE,
    priority_policy VARCHAR(32) NOT NULL DEFAULT 'user_first',
    risk_notes TEXT NOT NULL DEFAULT '',
    gateway_dispatch_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    gateway_shadow_mode BOOLEAN NOT NULL DEFAULT TRUE,
    gateway_traffic_percent DECIMAL(8,4) NOT NULL DEFAULT 0,
    gateway_allowed_group_ids TEXT NOT NULL DEFAULT '',
    gateway_allowed_models TEXT NOT NULL DEFAULT '',
    gateway_min_remain_ratio DECIMAL(8,4) NOT NULL DEFAULT 0.1000,
    gateway_max_daily_quota DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT carpool_revenue_configs_singleton CHECK (id = 1),
    CONSTRAINT carpool_revenue_configs_ratio_check CHECK (user_share_ratio >= 0 AND platform_share_ratio >= 0),
    CONSTRAINT carpool_revenue_configs_min_withdraw_check CHECK (min_withdraw_amount >= 0)
);

INSERT INTO carpool_revenue_configs (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_dispatch_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_shadow_mode BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_traffic_percent DECIMAL(8,4) NOT NULL DEFAULT 0;
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_allowed_group_ids TEXT NOT NULL DEFAULT '';
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_allowed_models TEXT NOT NULL DEFAULT '';
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_min_remain_ratio DECIMAL(8,4) NOT NULL DEFAULT 0.1000;
ALTER TABLE carpool_revenue_configs ADD COLUMN IF NOT EXISTS gateway_max_daily_quota DECIMAL(20,8) NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS carpool_revenue_contributions (
    id BIGSERIAL PRIMARY KEY,
    participant_id BIGINT NOT NULL REFERENCES carpool_participants(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id BIGINT NOT NULL REFERENCES carpool_sessions(id) ON DELETE CASCADE,
    vehicle_type_id BIGINT NOT NULL REFERENCES carpool_vehicle_types(id) ON DELETE RESTRICT,
    subscription_id BIGINT NULL REFERENCES user_subscriptions(id) ON DELETE SET NULL,
    subscription_group_id BIGINT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    enabled_at TIMESTAMPTZ NULL,
    disabled_at TIMESTAMPTZ NULL,
    share_ratio DECIMAL(8,4) NOT NULL DEFAULT 0.7000,
    status VARCHAR(32) NOT NULL DEFAULT 'disabled',
    paused_reason TEXT NOT NULL DEFAULT '',
    last_settled_at TIMESTAMPTZ NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT carpool_revenue_contributions_participant_unique UNIQUE (participant_id)
);

CREATE INDEX IF NOT EXISTS idx_carpool_revenue_contributions_user ON carpool_revenue_contributions (user_id);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_contributions_session ON carpool_revenue_contributions (session_id);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_contributions_status ON carpool_revenue_contributions (status);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_contributions_enabled ON carpool_revenue_contributions (enabled);

CREATE TABLE IF NOT EXISTS carpool_revenue_records (
    id BIGSERIAL PRIMARY KEY,
    contribution_id BIGINT NOT NULL REFERENCES carpool_revenue_contributions(id) ON DELETE CASCADE,
    participant_id BIGINT NOT NULL REFERENCES carpool_participants(id) ON DELETE CASCADE,
    session_id BIGINT NOT NULL REFERENCES carpool_sessions(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_group_id BIGINT NOT NULL DEFAULT 0,
    subscription_id BIGINT NULL REFERENCES user_subscriptions(id) ON DELETE SET NULL,
    api_key_id BIGINT NULL REFERENCES api_keys(id) ON DELETE SET NULL,
    usage_log_id BIGINT NULL REFERENCES usage_logs(id) ON DELETE SET NULL,
    request_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    request_api_key_id BIGINT NULL REFERENCES api_keys(id) ON DELETE SET NULL,
    request_account_id BIGINT NULL REFERENCES accounts(id) ON DELETE SET NULL,
    request_group_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL,
    dispatch_mode VARCHAR(32) NOT NULL DEFAULT '',
    decision_reason TEXT NOT NULL DEFAULT '',
    request_id VARCHAR(128) NOT NULL DEFAULT '',
    request_count INTEGER NOT NULL DEFAULT 0,
    quota_cost DECIMAL(20,10) NOT NULL DEFAULT 0,
    quota_before DECIMAL(20,10) NULL,
    quota_after DECIMAL(20,10) NULL,
    gross_revenue DECIMAL(20,8) NOT NULL DEFAULT 0,
    upstream_cost DECIMAL(20,8) NOT NULL DEFAULT 0,
    net_revenue DECIMAL(20,8) NOT NULL DEFAULT 0,
    user_share_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    platform_share_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    settlement_period VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'settled',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settled_at TIMESTAMPTZ NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS subscription_id BIGINT NULL REFERENCES user_subscriptions(id) ON DELETE SET NULL;
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS request_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS request_api_key_id BIGINT NULL REFERENCES api_keys(id) ON DELETE SET NULL;
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS request_account_id BIGINT NULL REFERENCES accounts(id) ON DELETE SET NULL;
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS request_group_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL;
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS dispatch_mode VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE carpool_revenue_records ADD COLUMN IF NOT EXISTS decision_reason TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_carpool_revenue_records_contribution ON carpool_revenue_records (contribution_id);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_records_user ON carpool_revenue_records (user_id);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_records_session ON carpool_revenue_records (session_id);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_records_status ON carpool_revenue_records (status);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_records_occurred_at ON carpool_revenue_records (occurred_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_carpool_revenue_records_request_unique
    ON carpool_revenue_records (contribution_id, request_id)
    WHERE request_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_carpool_revenue_records_usage_unique
    ON carpool_revenue_records (usage_log_id)
    WHERE usage_log_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS carpool_revenue_withdrawals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    participant_id BIGINT NULL REFERENCES carpool_participants(id) ON DELETE SET NULL,
    session_id BIGINT NULL REFERENCES carpool_sessions(id) ON DELETE SET NULL,
    amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    available_before DECIMAL(20,8) NOT NULL DEFAULT 0,
    available_after DECIMAL(20,8) NOT NULL DEFAULT 0,
    balance_before DECIMAL(20,8) NULL,
    balance_after DECIMAL(20,8) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'completed',
    balance_record_id BIGINT NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ NULL,
    failure_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_carpool_revenue_withdrawals_user ON carpool_revenue_withdrawals (user_id);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_withdrawals_session ON carpool_revenue_withdrawals (session_id);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_withdrawals_status ON carpool_revenue_withdrawals (status);
CREATE INDEX IF NOT EXISTS idx_carpool_revenue_withdrawals_requested_at ON carpool_revenue_withdrawals (requested_at);

UPDATE carpool_notice_versions
SET content_md = REPLACE(
    content_md,
    '4. 中转投入计划属于第二阶段能力，必须经过车内成员正式投票确认后才会执行。',
    '4. 中转投入计划属于后续扩展能力；如车类型支持，发车后用户可自主选择是否将自己的订阅额度投入中转，不影响其他成员。'
)
WHERE active = TRUE
  AND content_md LIKE '%中转投入计划属于第二阶段能力，必须经过车内成员正式投票确认后才会执行%';
