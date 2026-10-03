-- +goose Up
-- +goose StatementBegin

-- Promokod qamrovi:
--   user    - aynan bitta foydalanuvchi uchun (target_user_id), 1 marta
--   limited - ko'pi bilan max_uses ta foydalanuvchi uchun (har biri 1 martadan)
--   all     - hamma uchun (cheklovsiz, har bir foydalanuvchi 1 martadan)
-- Eski promokodlar "birinchi kelgan 1 marta ishlatadi" edi = limited, max_uses=1.
ALTER TABLE promo_codes ADD COLUMN IF NOT EXISTS scope VARCHAR(10) NOT NULL DEFAULT 'limited'
    CHECK (scope IN ('user', 'limited', 'all'));
ALTER TABLE promo_codes ADD COLUMN IF NOT EXISTS target_user_id UUID REFERENCES users(id) DEFAULT NULL;
ALTER TABLE promo_codes ADD COLUMN IF NOT EXISTS max_uses INT DEFAULT 1;
ALTER TABLE promo_codes ADD COLUMN IF NOT EXISTS used_count INT NOT NULL DEFAULT 0;

UPDATE promo_codes SET used_count = 1 WHERE status = 'used' AND used_by IS NOT NULL;

CREATE TABLE IF NOT EXISTS promo_code_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promo_code_id UUID NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    coins INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (promo_code_id, user_id)
);

-- Eski ishlatilgan kodlarni yangi jadvalga ko'chiramiz ("Mening promokodlarim" shu jadvaldan o'qiydi).
INSERT INTO promo_code_redemptions (promo_code_id, user_id, coins, created_at)
SELECT id, used_by, value, COALESCE(used_at, now()) FROM promo_codes WHERE used_by IS NOT NULL
ON CONFLICT DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_promo_redemptions_user ON promo_code_redemptions(user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS promo_code_redemptions;
ALTER TABLE promo_codes DROP COLUMN IF EXISTS used_count;
ALTER TABLE promo_codes DROP COLUMN IF EXISTS max_uses;
ALTER TABLE promo_codes DROP COLUMN IF EXISTS target_user_id;
ALTER TABLE promo_codes DROP COLUMN IF EXISTS scope;

-- +goose StatementEnd
