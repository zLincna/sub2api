ALTER TABLE lottery_prizes
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

ALTER TABLE lottery_draw_records
    ADD COLUMN IF NOT EXISTS prize_description TEXT NOT NULL DEFAULT '';

UPDATE lottery_draw_records r
SET prize_description = COALESCE(p.description, '')
FROM lottery_prizes p
WHERE r.prize_id = p.id
  AND COALESCE(r.prize_description, '') = '';

CREATE INDEX IF NOT EXISTS idx_lottery_draw_records_user_created_at
    ON lottery_draw_records (user_id, created_at DESC);
