ALTER TABLE products
    ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', name), 'A') ||
        setweight(to_tsvector('english', description), 'C')
    ) STORED;

CREATE INDEX products_search_vector_idx ON products USING gin (search_vector) WHERE deleted_at IS NULL;
CREATE INDEX products_name_trgm_idx ON products USING gin (name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE INDEX products_sku_prefix_idx ON products (sku text_pattern_ops) WHERE deleted_at IS NULL;
CREATE INDEX products_in_stock_idx ON products (created_at DESC) WHERE deleted_at IS NULL AND stock > 0;
