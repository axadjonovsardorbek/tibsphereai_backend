-- +goose Up
-- +goose StatementBegin

-- 00011 dagi 6 ta mustaqil son (xp/coin har bir daraja uchun alohida) o'rniga
-- soddalashtirildi: "easy" uchun bitta bazaviy XP/Coin, "medium"/"hard" esa shu
-- bazaga koeffitsient (multiplier) bilan ko'paytiriladi (AppSettingService.DifficultyReward).
DELETE FROM app_settings WHERE key IN (
	'xp_reward_easy', 'coin_reward_easy',
	'xp_reward_medium', 'coin_reward_medium',
	'xp_reward_hard', 'coin_reward_hard'
);

INSERT INTO app_settings (key, value) VALUES ('base_xp_reward', '50') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('base_coin_reward', '5') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('medium_reward_multiplier', '1.5') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('hard_reward_multiplier', '2') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM app_settings WHERE key IN (
	'base_xp_reward', 'base_coin_reward',
	'medium_reward_multiplier', 'hard_reward_multiplier'
);

INSERT INTO app_settings (key, value) VALUES ('xp_reward_easy', '50') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('coin_reward_easy', '5') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('xp_reward_medium', '100') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('coin_reward_medium', '10') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('xp_reward_hard', '150') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('coin_reward_hard', '15') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd
