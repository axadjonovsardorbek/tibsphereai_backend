-- +goose Up
-- +goose StatementBegin

-- Bitta simulyatsiya sessiyasida foydalanuvchi shu case'ga nechta xabar (chat/javob)
-- yubora olishi. Limitga yetsa sessiya avtomatik yakunlanadi. Default 10.
ALTER TABLE cases ADD COLUMN IF NOT EXISTS max_attempts INT NOT NULL DEFAULT 10;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE cases DROP COLUMN IF EXISTS max_attempts;

-- +goose StatementEnd
