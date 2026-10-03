-- +goose Up
-- +goose StatementBegin

-- Promokod hamkorga bog'lanishi shart emas (admin hamkorsiz ham yarata oladi).
ALTER TABLE promo_codes ALTER COLUMN partner_id DROP NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DELETE FROM promo_codes WHERE partner_id IS NULL;
ALTER TABLE promo_codes ALTER COLUMN partner_id SET NOT NULL;

-- +goose StatementEnd
