CREATE TABLE IF NOT EXISTS carpool_vehicle_types (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    seat_count INTEGER NOT NULL DEFAULT 2,
    total_price DECIMAL(20,2) NOT NULL DEFAULT 0,
    unit_price DECIMAL(20,2) NOT NULL DEFAULT 0,
    service_days INTEGER NOT NULL DEFAULT 30,
    refund_wait_hours INTEGER NOT NULL DEFAULT 2,
    completed_base_count INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    support_revenue_pool BOOLEAN NOT NULL DEFAULT FALSE,
    require_static_ip BOOLEAN NOT NULL DEFAULT TRUE,
    wait_duration_options JSONB NULL DEFAULT '[2, 6, 12, 24]'::jsonb,
    refund_methods JSONB NULL DEFAULT '["balance", "gateway"]'::jsonb,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS carpool_sessions (
    id BIGSERIAL PRIMARY KEY,
    vehicle_type_id BIGINT NOT NULL REFERENCES carpool_vehicle_types(id) ON DELETE CASCADE,
    session_no VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'recruiting',
    seat_count INTEGER NOT NULL DEFAULT 2,
    paid_count INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NULL,
    filled_at TIMESTAMPTZ NULL,
    provisioned_at TIMESTAMPTZ NULL,
    service_started_at TIMESTAMPTZ NULL,
    service_ended_at TIMESTAMPTZ NULL,
    account_info JSONB NULL DEFAULT '{}'::jsonb,
    proxy_info JSONB NULL DEFAULT '{}'::jsonb,
    communication JSONB NULL DEFAULT '{}'::jsonb,
    admin_notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS carpool_notice_versions (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(120) NOT NULL,
    content_md TEXT NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 1,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS carpool_participants (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NULL REFERENCES carpool_sessions(id) ON DELETE SET NULL,
    vehicle_type_id BIGINT NOT NULL REFERENCES carpool_vehicle_types(id) ON DELETE RESTRICT,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payment_order_id BIGINT NULL REFERENCES payment_orders(id) ON DELETE SET NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_payment',
    amount DECIMAL(20,2) NOT NULL DEFAULT 0,
    wait_until TIMESTAMPTZ NOT NULL,
    refund_method VARCHAR(32) NOT NULL DEFAULT 'balance',
    notice_version_id BIGINT NULL REFERENCES carpool_notice_versions(id) ON DELETE SET NULL,
    notice_accepted_at TIMESTAMPTZ NULL,
    notice_accept_ip VARCHAR(64) NOT NULL DEFAULT '',
    joined_at TIMESTAMPTZ NULL,
    paid_at TIMESTAMPTZ NULL,
    refunded_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS carpool_vouchers (
    id BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES carpool_sessions(id) ON DELETE CASCADE,
    file_url TEXT NOT NULL DEFAULT '',
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NULL,
    uploaded_by BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_carpool_vehicle_types_enabled ON carpool_vehicle_types (enabled);
CREATE INDEX IF NOT EXISTS idx_carpool_vehicle_types_sort_order ON carpool_vehicle_types (sort_order);
CREATE INDEX IF NOT EXISTS idx_carpool_vehicle_types_seat_count ON carpool_vehicle_types (seat_count);
CREATE INDEX IF NOT EXISTS idx_carpool_sessions_vehicle_type_id ON carpool_sessions (vehicle_type_id);
CREATE INDEX IF NOT EXISTS idx_carpool_sessions_status ON carpool_sessions (status);
CREATE INDEX IF NOT EXISTS idx_carpool_sessions_created_at ON carpool_sessions (created_at);
CREATE INDEX IF NOT EXISTS idx_carpool_sessions_vehicle_status ON carpool_sessions (vehicle_type_id, status);
CREATE INDEX IF NOT EXISTS idx_carpool_notice_versions_active ON carpool_notice_versions (active);
CREATE INDEX IF NOT EXISTS idx_carpool_notice_versions_version ON carpool_notice_versions (version);
CREATE INDEX IF NOT EXISTS idx_carpool_participants_session_id ON carpool_participants (session_id);
CREATE INDEX IF NOT EXISTS idx_carpool_participants_vehicle_type_id ON carpool_participants (vehicle_type_id);
CREATE INDEX IF NOT EXISTS idx_carpool_participants_user_id ON carpool_participants (user_id);
CREATE INDEX IF NOT EXISTS idx_carpool_participants_payment_order_id ON carpool_participants (payment_order_id);
CREATE INDEX IF NOT EXISTS idx_carpool_participants_status ON carpool_participants (status);
CREATE INDEX IF NOT EXISTS idx_carpool_participants_wait_until ON carpool_participants (wait_until);
CREATE INDEX IF NOT EXISTS idx_carpool_vouchers_session_id ON carpool_vouchers (session_id);
CREATE INDEX IF NOT EXISTS idx_carpool_vouchers_created_at ON carpool_vouchers (created_at);

INSERT INTO carpool_notice_versions (title, content_md, version, active, published_at)
SELECT '拼车用户须知', '# 拼车用户须知

请在支付前完整阅读并确认以下规则：

1. 拼车为多人共同等待成团，人满后由管理员采购和交付。
2. 每种车会配置固定可退款时间；到达该时间后，您可在“我的拼车”中自行发起退款，未发起则继续等待成团。
3. 发车后的账号、代理、使用方式和沟通方式以管理员交付信息为准。
4. 中转投入计划属于第二阶段能力，必须经过车内成员正式投票确认后才会执行。
5. 请勿将车内敏感信息泄露给非本车成员。
', 1, TRUE, NOW()
WHERE NOT EXISTS (SELECT 1 FROM carpool_notice_versions WHERE active = TRUE);

INSERT INTO carpool_vehicle_types (
    name,
    seat_count,
    total_price,
    unit_price,
    service_days,
    refund_wait_hours,
    completed_base_count,
    enabled,
    support_revenue_pool,
    require_static_ip,
    wait_duration_options,
    refund_methods,
    description,
    sort_order
)
SELECT
    seat_count || '人车',
    seat_count,
    1300.00,
    ROUND((1300.00 / seat_count)::numeric, 2),
    30,
    2,
    0,
    FALSE,
    TRUE,
    TRUE,
    '[2, 6, 12, 24]'::jsonb,
    '["balance", "gateway"]'::jsonb,
    'Codex Pro 20x 拼车，1号1个静态住宅IP，满员后按顺序发车。',
    seat_count * 10
FROM generate_series(1, 10) AS seat_count
WHERE NOT EXISTS (SELECT 1 FROM carpool_vehicle_types);
