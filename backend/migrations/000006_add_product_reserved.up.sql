ALTER TABLE products ADD COLUMN reserved integer NOT NULL DEFAULT 0 CHECK (reserved >= 0);

UPDATE products p
SET stock = p.stock + r.quantity, reserved = r.quantity
FROM (
    SELECT oi.product_id, sum(oi.quantity) AS quantity
    FROM order_items oi
    JOIN orders o ON o.id = oi.order_id
    WHERE o.status = 'PENDING'
    GROUP BY oi.product_id
) r
WHERE p.id = r.product_id;

ALTER TABLE products ADD CONSTRAINT products_reserved_within_stock CHECK (reserved <= stock);

DROP INDEX products_in_stock_idx;
CREATE INDEX products_in_stock_idx ON products (created_at DESC) WHERE deleted_at IS NULL AND stock > reserved;
