-- +goose Up
-- +goose StatementBegin

-- categories.audience - "Shifokorlar rejimi" (doctor) va "Birinchi yordam /
-- Aholi rejimi" (citizen) o'rtasida dinamik ajratish uchun (frontend home
-- ekranidagi rejim toggle shu bo'yicha bo'lim ro'yxatini filtrlaydi).
ALTER TABLE categories ADD COLUMN IF NOT EXISTS audience VARCHAR(20) NOT NULL DEFAULT 'doctor'
	CHECK (audience IN ('doctor', 'citizen'));

CREATE INDEX IF NOT EXISTS idx_categories_audience ON categories(audience) WHERE deleted_at = 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_categories_audience;
ALTER TABLE categories DROP COLUMN IF EXISTS audience;

-- +goose StatementEnd
