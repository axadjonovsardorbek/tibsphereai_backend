-- +goose Up
-- +goose StatementBegin

-- Adminlar CRUD (ro'yxat/qo'shish/yangilash/o'chirish) uchun soft-delete qo'shamiz,
-- boshqa jadvallardagi kabi (deleted_at BIGINT, 0 = o'chirilmagan).
ALTER TABLE admins ADD COLUMN IF NOT EXISTS deleted_at BIGINT DEFAULT 0;

-- login UNIQUE cheklovi soft-delete bilan mos kelishi uchun (o'chirilgan admin login'ini
-- qayta ishlatish mumkin bo'lsin), boshqa jadvallardagi UNIQUE(x, deleted_at) patterni.
ALTER TABLE admins DROP CONSTRAINT IF EXISTS admins_login_key;
ALTER TABLE admins ADD CONSTRAINT admins_login_deleted_key UNIQUE (login, deleted_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE admins DROP CONSTRAINT IF EXISTS admins_login_deleted_key;
ALTER TABLE admins ADD CONSTRAINT admins_login_key UNIQUE (login);
ALTER TABLE admins DROP COLUMN IF EXISTS deleted_at;

-- +goose StatementEnd
