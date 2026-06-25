ALTER TABLE carpool_vehicle_types
    ADD COLUMN IF NOT EXISTS product VARCHAR(32) NOT NULL DEFAULT 'openai',
    ADD COLUMN IF NOT EXISTS plan_tier VARCHAR(32) NOT NULL DEFAULT 'pro',
    ADD COLUMN IF NOT EXISTS multiplier VARCHAR(32) NOT NULL DEFAULT '20x',
    ADD COLUMN IF NOT EXISTS refund_wait_hours INTEGER NOT NULL DEFAULT 2,
    ADD COLUMN IF NOT EXISTS completed_base_count INTEGER NOT NULL DEFAULT 0;

UPDATE carpool_vehicle_types
SET
    product = COALESCE(NULLIF(product, ''), 'openai'),
    plan_tier = COALESCE(NULLIF(plan_tier, ''), 'pro'),
    multiplier = COALESCE(NULLIF(multiplier, ''), '20x')
WHERE product = '' OR plan_tier = '' OR multiplier = '';

CREATE INDEX IF NOT EXISTS idx_carpool_vehicle_types_product_plan ON carpool_vehicle_types (product, plan_tier, multiplier);
