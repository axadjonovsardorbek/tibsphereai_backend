-- +goose Up
-- +goose StatementBegin

-- Kunlik bepul limit tugagan (obunasiz) foydalanuvchi 1 ta case'ni tanga evaziga
-- boshlashi mumkin. Narx (tanga) admin panelda sozlanadi; 0 - tanga bilan ochish o'chiq.
INSERT INTO app_settings (key, value) VALUES ('case_coin_cost', '5') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM app_settings WHERE key = 'case_coin_cost';

-- +goose StatementEnd
