ALTER TABLE carpool_vehicle_types
    ADD COLUMN IF NOT EXISTS refund_wait_hours INTEGER NOT NULL DEFAULT 2,
    ADD COLUMN IF NOT EXISTS completed_base_count INTEGER NOT NULL DEFAULT 0;

UPDATE carpool_vehicle_types
SET
    refund_wait_hours = CASE
        WHEN refund_wait_hours IS NULL OR refund_wait_hours <= 0 THEN 2
        ELSE refund_wait_hours
    END,
    completed_base_count = CASE
        WHEN completed_base_count IS NULL OR completed_base_count < 0 THEN 0
        ELSE completed_base_count
    END;
