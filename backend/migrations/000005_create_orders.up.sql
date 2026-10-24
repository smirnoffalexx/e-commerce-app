CREATE TABLE orders (
    id              uuid          PRIMARY KEY,
    user_id         uuid          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    customer_email  text          NOT NULL,
    status          text          NOT NULL CHECK (status IN ('PENDING', 'PAID', 'FAILED')),
    total           numeric(16,2) NOT NULL CHECK (total >= 0),
    idempotency_key text          NOT NULL CHECK (char_length(idempotency_key) BETWEEN 1 AND 255),
    request_hash    text          NOT NULL,
    card_last4      text          NOT NULL CHECK (card_last4 ~ '^[0-9]{4}$'),
    payment_id      text,
    failure_code    text,
    paid_at         timestamptz,
    created_at      timestamptz   NOT NULL DEFAULT now(),
    updated_at      timestamptz   NOT NULL DEFAULT now(),
    CHECK ((status = 'PAID') = (payment_id IS NOT NULL AND paid_at IS NOT NULL)),
    CHECK ((status = 'FAILED') = (failure_code IS NOT NULL))
);

CREATE UNIQUE INDEX orders_user_idempotency_key ON orders (user_id, idempotency_key);
CREATE INDEX orders_user_created_at_idx ON orders (user_id, created_at DESC, id DESC);
CREATE INDEX orders_created_at_idx ON orders (created_at DESC, id DESC);
CREATE INDEX orders_pending_idx ON orders (created_at) WHERE status = 'PENDING';

CREATE TABLE order_items (
    order_id   uuid          NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    line_no    smallint      NOT NULL CHECK (line_no >= 1),
    product_id uuid          NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    sku        text          NOT NULL,
    name       text          NOT NULL,
    unit_price numeric(12,2) NOT NULL CHECK (unit_price >= 0),
    quantity   integer       NOT NULL CHECK (quantity BETWEEN 1 AND 100),
    line_total numeric(16,2) NOT NULL CHECK (line_total = unit_price * quantity),
    PRIMARY KEY (order_id, line_no)
);

CREATE INDEX order_items_product_id_idx ON order_items (product_id);
