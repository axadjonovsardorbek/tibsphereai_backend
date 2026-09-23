-- +goose Up
-- +goose StatementBegin

-- case_to_uzs_rate hech qanday kodda ishlatilmaydi - faqat case_to_coin_rate
-- (CategoryRepo.Create fallback) real ishlaydi. Kerak emas, olib tashlandi.
DELETE FROM app_settings WHERE key = 'case_to_uzs_rate';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

INSERT INTO app_settings (key, value) VALUES ('case_to_uzs_rate', '500') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd
