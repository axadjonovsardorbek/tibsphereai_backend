-- +goose Up
-- +goose StatementBegin

-- Case qiyinlik darajasi (difficulty) bo'yicha bazaviy XP/Coin mukofoti - admin
-- "Sozlamalar -> Mukofot sozlamalari" orqali o'zgartira oladi (AppSettingService.DifficultyReward).
-- Avval xp=health_percent, coins=health_percent/10 edi (faqat sog'liqqa qarab, case
-- qanchalik qiyin bo'lishidan qat'i nazar). Endi yakuniy mukofot shu bazaga
-- health_percent nisbatan mutanosib beriladi.
INSERT INTO app_settings (key, value) VALUES ('xp_reward_easy', '50') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('coin_reward_easy', '5') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('xp_reward_medium', '100') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('coin_reward_medium', '10') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('xp_reward_hard', '150') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('coin_reward_hard', '15') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM app_settings WHERE key IN (
	'xp_reward_easy', 'coin_reward_easy',
	'xp_reward_medium', 'coin_reward_medium',
	'xp_reward_hard', 'coin_reward_hard'
);

-- +goose StatementEnd
