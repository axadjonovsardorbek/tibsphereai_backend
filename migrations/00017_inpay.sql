-- +goose Up
-- +goose StatementBegin

-- inPAY (https://inpay.uz/api/) to'lov provayderi: order_id'ni inPAY o'zi generatsiya qiladi,
-- webhook shu id bilan keladi, shuning uchun bizning orders.id bilan bog'laymiz.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_type_check;
ALTER TABLE orders ADD CONSTRAINT orders_payment_type_check
    CHECK (payment_type IN ('click', 'payme', 'inpay', 'coins', 'admin'));
ALTER TABLE orders ADD COLUMN IF NOT EXISTS inpay_order_id VARCHAR(64) DEFAULT NULL;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS inpay_transaction_id BIGINT DEFAULT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS orders_inpay_order_id_key ON orders (inpay_order_id) WHERE inpay_order_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS orders_inpay_order_id_key;
ALTER TABLE orders DROP COLUMN IF EXISTS inpay_transaction_id;
ALTER TABLE orders DROP COLUMN IF EXISTS inpay_order_id;
UPDATE orders SET payment_type = 'click' WHERE payment_type = 'inpay';
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_type_check;
ALTER TABLE orders ADD CONSTRAINT orders_payment_type_check
    CHECK (payment_type IN ('click', 'payme', 'coins', 'admin'));

-- +goose StatementEnd
