-- +goose Up
-- +goose StatementBegin

-- Foydalanuvchi darajasi (Level 1/2/3...) bo'yicha case yakunlaganda oladigan
-- bazaviy XP/Coin - endi app_settings'da emas, har bir level o'zining qiymatini
-- olib yuradi (admin "Levellar" sahifasida tahrirlaydi). Case qiyinlik darajasi
-- (easy/medium/hard) koeffitsienti hamon app_settings'da qoladi
-- (medium_reward_multiplier/hard_reward_multiplier). Ko'ring [[SimulationService.Finish]].
ALTER TABLE levels ADD COLUMN IF NOT EXISTS xp_reward INT NOT NULL DEFAULT 50;
ALTER TABLE levels ADD COLUMN IF NOT EXISTS coin_reward INT NOT NULL DEFAULT 5;

UPDATE levels SET xp_reward = 30, coin_reward = 3 WHERE level_number = 1;
UPDATE levels SET xp_reward = 50, coin_reward = 5 WHERE level_number = 2;
UPDATE levels SET xp_reward = 75, coin_reward = 8 WHERE level_number = 3;
UPDATE levels SET xp_reward = 100, coin_reward = 10 WHERE level_number = 4;
UPDATE levels SET xp_reward = 150, coin_reward = 15 WHERE level_number = 5;

-- 00012'da qo'shilgan base_xp_reward/base_coin_reward endi kerak emas (o'rnini
-- level bo'yicha xp_reward/coin_reward bosdi) - faqat koeffitsientlar qoladi.
DELETE FROM app_settings WHERE key IN ('base_xp_reward', 'base_coin_reward');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

INSERT INTO app_settings (key, value) VALUES ('base_xp_reward', '50') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('base_coin_reward', '5') ON CONFLICT (key) DO NOTHING;

ALTER TABLE levels DROP COLUMN IF EXISTS xp_reward;
ALTER TABLE levels DROP COLUMN IF EXISTS coin_reward;

-- +goose StatementEnd
