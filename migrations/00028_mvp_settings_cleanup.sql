-- +goose Up
-- +goose StatementBegin

-- TZ_MVP sozlamalar tozalash:
--  * coin_to_uzs_rate [Kerak emas]: tanga paketlari tariflarda sotiladi, tangani so'mga aylantirib
--    qisman to'lash olib tashlandi.
--  * voice_enabled [Keyinga qoldiriladi]: to'liq ovozli interaktiv chat keyingi bosqichda; hozircha o'chiq.
--  * practice_images_enabled: AI tibbiy tasvirlar (default o'chiq, admin yoqadi).
DELETE FROM app_settings WHERE key = 'coin_to_uzs_rate';
INSERT INTO app_settings (key, value) VALUES ('voice_enabled', 'false')
    ON CONFLICT (key) DO UPDATE SET value = 'false', updated_at = now();
INSERT INTO app_settings (key, value) VALUES ('practice_images_enabled', 'false') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM app_settings WHERE key = 'practice_images_enabled';
INSERT INTO app_settings (key, value) VALUES ('coin_to_uzs_rate', '100') ON CONFLICT (key) DO NOTHING;
UPDATE app_settings SET value = 'true', updated_at = now() WHERE key = 'voice_enabled';

-- +goose StatementEnd
