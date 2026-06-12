-- Add phone registration metadata.
-- phone is stored normalized as mainland China digits (11 chars) when provided.

ALTER TABLE users ADD COLUMN IF NOT EXISTS phone varchar(32) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_verified_at timestamptz NULL;

CREATE UNIQUE INDEX IF NOT EXISTS users_phone_unique_active
    ON users(phone)
    WHERE deleted_at IS NULL AND phone <> '';

COMMENT ON COLUMN users.phone IS '注册手机号，归一化为大陆手机号数字；空字符串表示未填写。';
COMMENT ON COLUMN users.phone_verified_at IS '手机号短信验证通过时间；NULL 表示未验证。';
