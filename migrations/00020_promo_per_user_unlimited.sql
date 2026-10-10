-- +goose Up
-- +goose StatementBegin

-- max_uses_per_user = 0 - bitta foydalanuvchi uchun cheksiz.
ALTER TABLE promo_codes DROP CONSTRAINT IF EXISTS promo_codes_max_uses_per_user_check;
ALTER TABLE promo_codes ADD CONSTRAINT promo_codes_max_uses_per_user_check CHECK (max_uses_per_user >= 0);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

UPDATE promo_codes SET max_uses_per_user = 1 WHERE max_uses_per_user = 0;
ALTER TABLE promo_codes DROP CONSTRAINT IF EXISTS promo_codes_max_uses_per_user_check;
ALTER TABLE promo_codes ADD CONSTRAINT promo_codes_max_uses_per_user_check CHECK (max_uses_per_user >= 1);

-- +goose StatementEnd
