-- +goose Up
-- +goose StatementBegin

-- Promokodni bitta foydalanuvchi necha marta ishlata olishi (default 1 - eski xatti-harakat).
-- max_uses endi JAMI ishlatishlar soni (har bir foydalanuvchining takroriy ishlatishi ham sanaladi).
ALTER TABLE promo_codes ADD COLUMN IF NOT EXISTS max_uses_per_user INT NOT NULL DEFAULT 1;
ALTER TABLE promo_codes ADD CONSTRAINT promo_codes_max_uses_per_user_check CHECK (max_uses_per_user >= 1);

-- Bir foydalanuvchi bir kodni bir necha marta ishlata olishi uchun UNIQUE olib tashlanadi.
ALTER TABLE promo_code_redemptions DROP CONSTRAINT IF EXISTS promo_code_redemptions_promo_code_id_user_id_key;
CREATE INDEX IF NOT EXISTS idx_promo_redemptions_code_user ON promo_code_redemptions(promo_code_id, user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_promo_redemptions_code_user;
-- Takroriy yozuvlar bo'lsa, eng eskisi qoladi (UNIQUE'ni qayta qo'yish uchun).
DELETE FROM promo_code_redemptions a USING promo_code_redemptions b
WHERE a.promo_code_id = b.promo_code_id AND a.user_id = b.user_id AND a.created_at > b.created_at;
ALTER TABLE promo_code_redemptions ADD CONSTRAINT promo_code_redemptions_promo_code_id_user_id_key UNIQUE (promo_code_id, user_id);
ALTER TABLE promo_codes DROP CONSTRAINT IF EXISTS promo_codes_max_uses_per_user_check;
ALTER TABLE promo_codes DROP COLUMN IF EXISTS max_uses_per_user;

-- +goose StatementEnd
