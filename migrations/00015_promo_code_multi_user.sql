-- +goose Up
-- +goose StatementBegin

-- scope='users': aniq ko'rsatilgan bir nechta foydalanuvchi uchun (promo_code_targets ro'yxati).
ALTER TABLE promo_codes DROP CONSTRAINT IF EXISTS promo_codes_scope_check;
ALTER TABLE promo_codes ADD CONSTRAINT promo_codes_scope_check
    CHECK (scope IN ('user', 'users', 'limited', 'all'));

CREATE TABLE IF NOT EXISTS promo_code_targets (
    promo_code_id UUID NOT NULL REFERENCES promo_codes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    PRIMARY KEY (promo_code_id, user_id)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS promo_code_targets;
DELETE FROM promo_codes WHERE scope = 'users';
ALTER TABLE promo_codes DROP CONSTRAINT IF EXISTS promo_codes_scope_check;
ALTER TABLE promo_codes ADD CONSTRAINT promo_codes_scope_check CHECK (scope IN ('user', 'limited', 'all'));

-- +goose StatementEnd
