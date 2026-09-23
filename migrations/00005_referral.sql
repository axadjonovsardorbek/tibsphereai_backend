-- +goose Up
-- +goose StatementBegin

-- Referral tizimi: har bir user'ning o'ziga xos referral_code'i bor, do'stini
-- shu kod orqali taklif qilsa (registratsiyada referral_code yuboriladi),
-- ikkalasiga ham tanga beriladi (miqdor app_settings'da sozlanadi).
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_code VARCHAR(12);
ALTER TABLE users ADD COLUMN IF NOT EXISTS referred_by UUID REFERENCES users(id) DEFAULT NULL;

-- Mavjud userlarga ham noyob kod to'ldiramiz (id'ning boshidan olib).
UPDATE users SET referral_code = upper(substr(replace(id::text, '-', ''), 1, 8))
WHERE referral_code IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_referral_code ON users(referral_code);

CREATE TABLE IF NOT EXISTS referral_rewards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_id UUID NOT NULL REFERENCES users(id),
    -- referred_id UNIQUE - bitta user faqat bir marta "taklif qilingan" hisoblanadi
    referred_id UUID NOT NULL UNIQUE REFERENCES users(id),
    referrer_coins INT NOT NULL DEFAULT 0,
    referred_coins INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_referral_rewards_referrer ON referral_rewards(referrer_id);

INSERT INTO app_settings (key, value) VALUES ('referral_reward_referrer_coins', '50') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('referral_reward_referred_coins', '20') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS referral_rewards;
DROP INDEX IF EXISTS idx_users_referral_code;
ALTER TABLE users DROP COLUMN IF EXISTS referred_by;
ALTER TABLE users DROP COLUMN IF EXISTS referral_code;
DELETE FROM app_settings WHERE key IN ('referral_reward_referrer_coins', 'referral_reward_referred_coins');

-- +goose StatementEnd
