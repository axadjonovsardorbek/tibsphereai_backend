-- +goose Up
-- +goose StatementBegin

-- Dinamik token-to-coin to'lov: AI sarflagan real tokenlar admin belgilagan
-- "1 000 token = X tanga" tarifi bo'yicha foydalanuvchi balansidan yechiladi.
-- coin_billed - sessiya tanga hisobidan to'lanadi (kunlik limit tugagach use_coins bilan boshlangan).
ALTER TABLE simulation_sessions ADD COLUMN IF NOT EXISTS prompt_tokens INT NOT NULL DEFAULT 0;
ALTER TABLE simulation_sessions ADD COLUMN IF NOT EXISTS completion_tokens INT NOT NULL DEFAULT 0;
ALTER TABLE simulation_sessions ADD COLUMN IF NOT EXISTS coin_billed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE simulation_sessions ADD COLUMN IF NOT EXISTS coins_charged INT NOT NULL DEFAULT 0;

INSERT INTO app_settings (key, value) VALUES ('coins_per_1k_tokens', '1') ON CONFLICT (key) DO NOTHING;

-- Yangi case'lar uchun default urinishlar soni 1.
ALTER TABLE cases ALTER COLUMN max_attempts SET DEFAULT 1;

-- Eski qat'iy "1 case = X tanga" modeli bekor qilindi.
DELETE FROM app_settings WHERE key = 'case_coin_cost';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE cases ALTER COLUMN max_attempts SET DEFAULT 10;
DELETE FROM app_settings WHERE key = 'coins_per_1k_tokens';
ALTER TABLE simulation_sessions DROP COLUMN IF EXISTS coins_charged;
ALTER TABLE simulation_sessions DROP COLUMN IF EXISTS coin_billed;
ALTER TABLE simulation_sessions DROP COLUMN IF EXISTS completion_tokens;
ALTER TABLE simulation_sessions DROP COLUMN IF EXISTS prompt_tokens;

-- +goose StatementEnd
