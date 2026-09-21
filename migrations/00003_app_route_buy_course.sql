-- +goose Up
-- +goose StatementBegin

-- storage/postgres/app_route.go va pkg/models/app_route.go "buy_course" ustunini kutadi,
-- lekin 00001_init_db.sql'da jadval yaratilganda bu ustun qo'shilmagan edi (schema/kod drift).
ALTER TABLE app_routes ADD COLUMN IF NOT EXISTS buy_course BOOLEAN NOT NULL DEFAULT FALSE;

-- storage/postgres/app_route.go (GetById/GetAll/Update/Delete) "deleted_at" ustunini ham
-- kutadi (boshqa jadvallardagi soft-delete pattern), lekin u ham yaratilmagan edi.
ALTER TABLE app_routes ADD COLUMN IF NOT EXISTS deleted_at BIGINT DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE app_routes DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE app_routes DROP COLUMN IF EXISTS buy_course;

-- +goose StatementEnd
