-- +goose Up
-- +goose StatementBegin

-- Platforma ovozli funksiyasi (case shikoyati/chat javoblarini frontend TTS orqali
-- o'qib berish) global yoqish/o'chirish uchun admin killswitch (AppSettingService.VoiceEnabled).
INSERT INTO app_settings (key, value) VALUES ('voice_enabled', 'true') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM app_settings WHERE key = 'voice_enabled';

-- +goose StatementEnd
