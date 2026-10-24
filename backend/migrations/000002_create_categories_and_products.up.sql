CREATE TABLE categories (
    id         uuid        PRIMARY KEY,
    name       text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 64),
    slug       text        NOT NULL CHECK (slug <> ''),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE UNIQUE INDEX categories_slug_active_key ON categories (slug) WHERE deleted_at IS NULL;

CREATE TABLE products (
    id          uuid          PRIMARY KEY,
    sku         text          NOT NULL CHECK (sku <> '' AND sku = upper(sku)),
    name        text          NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    description text          NOT NULL DEFAULT '' CHECK (char_length(description) <= 5000),
    category_id uuid          NOT NULL REFERENCES categories (id) ON DELETE RESTRICT,
    price       numeric(12,2) NOT NULL CHECK (price >= 0),
    stock       integer       NOT NULL CHECK (stock >= 0),
    weight_kg   numeric(10,3) NOT NULL CHECK (weight_kg >= 0),
    version     integer       NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at  timestamptz   NOT NULL DEFAULT now(),
    updated_at  timestamptz   NOT NULL DEFAULT now(),
    deleted_at  timestamptz
);

CREATE UNIQUE INDEX products_sku_active_key ON products (sku) WHERE deleted_at IS NULL;
CREATE INDEX products_category_id_idx ON products (category_id) WHERE deleted_at IS NULL;
CREATE INDEX products_created_at_idx ON products (created_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX products_price_idx ON products (price) WHERE deleted_at IS NULL;
