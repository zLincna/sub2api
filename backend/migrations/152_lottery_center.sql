CREATE TABLE IF NOT EXISTS lottery_prizes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    probability DECIMAL(10,6) NOT NULL DEFAULT 0,
    daily_stock INTEGER NOT NULL DEFAULT 0,
    daily_used INTEGER NOT NULL DEFAULT 0,
    total_stock INTEGER NOT NULL DEFAULT 0,
    total_used INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    color VARCHAR(32) NOT NULL DEFAULT '#f59e0b',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS lottery_chances (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_type VARCHAR(32) NOT NULL,
    source_key VARCHAR(128) NOT NULL,
    total_count INTEGER NOT NULL DEFAULT 0,
    used_count INTEGER NOT NULL DEFAULT 0,
    source_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    grant_date DATE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lottery_chances_user_source_key_unique UNIQUE (user_id, source_type, source_key)
);

CREATE TABLE IF NOT EXISTS lottery_draw_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chance_id BIGINT NOT NULL REFERENCES lottery_chances(id) ON DELETE CASCADE,
    prize_id BIGINT NOT NULL REFERENCES lottery_prizes(id) ON DELETE RESTRICT,
    prize_name VARCHAR(100) NOT NULL,
    amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    balance_before DECIMAL(20,8) NOT NULL DEFAULT 0,
    balance_after DECIMAL(20,8) NOT NULL DEFAULT 0,
    source_type VARCHAR(32) NOT NULL,
    config_snapshot JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lottery_prizes_enabled_sort ON lottery_prizes (enabled, sort_order);
CREATE INDEX IF NOT EXISTS idx_lottery_chances_user_id ON lottery_chances (user_id);
CREATE INDEX IF NOT EXISTS idx_lottery_chances_source_type ON lottery_chances (source_type);
CREATE INDEX IF NOT EXISTS idx_lottery_chances_expires_at ON lottery_chances (expires_at);
CREATE INDEX IF NOT EXISTS idx_lottery_chances_grant_date ON lottery_chances (grant_date);
CREATE INDEX IF NOT EXISTS idx_lottery_draw_records_user_id ON lottery_draw_records (user_id);
CREATE INDEX IF NOT EXISTS idx_lottery_draw_records_created_at ON lottery_draw_records (created_at);
CREATE INDEX IF NOT EXISTS idx_lottery_draw_records_source_type ON lottery_draw_records (source_type);

INSERT INTO lottery_prizes (name, amount, probability, daily_stock, total_stock, enabled, color, sort_order)
SELECT name, amount, probability, daily_stock, total_stock, enabled, color, sort_order
FROM (VALUES
    ('小确幸 $0.01', 0.01, 40.000000, 0, 0, TRUE, '#22c55e', 10),
    ('好运 $0.02', 0.02, 25.000000, 0, 0, TRUE, '#06b6d4', 20),
    ('惊喜 $0.05', 0.05, 18.000000, 0, 0, TRUE, '#3b82f6', 30),
    ('加餐 $0.10', 0.10, 10.000000, 0, 0, TRUE, '#8b5cf6', 40),
    ('手气王 $0.20', 0.20, 5.000000, 0, 0, TRUE, '#f59e0b', 50),
    ('锦鲤 $0.50', 0.50, 2.000000, 0, 0, TRUE, '#ef4444', 60)
) AS seed(name, amount, probability, daily_stock, total_stock, enabled, color, sort_order)
WHERE NOT EXISTS (SELECT 1 FROM lottery_prizes);
