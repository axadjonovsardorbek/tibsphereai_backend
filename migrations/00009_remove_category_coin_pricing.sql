-- +goose Up
-- +goose StatementBegin

-- "Bo'limda coin bilan narxlash" funksiyasi butunlay olib tashlandi: endi simulyatsiyaga
-- kirish faqat Premium obuna yoki kunlik bepul limit orqali (SimulationService.Start).
DROP TABLE IF EXISTS category_unlocks;
ALTER TABLE categories DROP COLUMN IF EXISTS coins_per_case;
DELETE FROM app_settings WHERE key = 'case_to_coin_rate';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE categories ADD COLUMN IF NOT EXISTS coins_per_case INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS category_unlocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    category_id UUID NOT NULL REFERENCES categories(id),
    cases_remaining INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, category_id)
);

INSERT INTO app_settings (key, value) VALUES ('case_to_coin_rate', '5') ON CONFLICT (key) DO NOTHING;

-- +goose StatementEnd
