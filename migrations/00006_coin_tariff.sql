-- +goose Up
-- +goose StatementBegin

-- tariffs.kind - 'subscription' (hozirgi Premium obuna, duration ishlatiladi) yoki
-- 'coin_package' (bir martalik tanga sotib olish, coins ishlatiladi, duration'ga
-- e'tibor berilmaydi). Order/to'lov (Click/Payme/gateway'siz) bir xil oqimdan o'tadi -
-- faqat to'lov PAID bo'lganda tariff.kind bo'yicha subscription yaratiladimi yoki
-- coins qo'shiladimi hal qilinadi (ko'ring OrderRepo.activateTariff).
ALTER TABLE tariffs ADD COLUMN IF NOT EXISTS kind VARCHAR(20) NOT NULL DEFAULT 'subscription'
	CHECK (kind IN ('subscription', 'coin_package'));
ALTER TABLE tariffs ADD COLUMN IF NOT EXISTS coins INT NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE tariffs DROP COLUMN IF EXISTS coins;
ALTER TABLE tariffs DROP COLUMN IF EXISTS kind;

-- +goose StatementEnd
