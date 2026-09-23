-- +goose Up
-- +goose StatementBegin

-- Yangi bo'lim (category) yaratilganda coins_per_case ko'rsatilmasa ishlatiladigan
-- global default'lar (CategoryRepo.Create). case_to_uzs_rate - shu default'ning
-- so'mdagi ma'nosi (coin_to_uzs_rate bilan mos: case_to_uzs_rate = case_to_coin_rate * coin_to_uzs_rate),
-- admin panelda narxni to'g'ri tushunish uchun ko'rsatma sifatida.
INSERT INTO app_settings (key, value) VALUES ('case_to_coin_rate', '5') ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('case_to_uzs_rate', '500') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM app_settings WHERE key IN ('case_to_coin_rate', 'case_to_uzs_rate');

-- +goose StatementEnd
