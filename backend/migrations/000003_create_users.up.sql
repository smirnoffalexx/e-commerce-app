CREATE TABLE users (
    id                       uuid        PRIMARY KEY,
    email                    text        NOT NULL CHECK (email <> '' AND email = lower(email)),
    password_hash            text        NOT NULL,
    role                     text        NOT NULL CHECK (role IN ('USER', 'ADMIN', 'SUPERADMIN')),
    is_active                boolean     NOT NULL DEFAULT true,
    must_change_password     boolean     NOT NULL DEFAULT false,
    temp_password_expires_at timestamptz,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now(),
    CHECK (NOT must_change_password OR temp_password_expires_at IS NOT NULL)
);

CREATE UNIQUE INDEX users_email_key ON users (email);
CREATE INDEX users_role_idx ON users (role);
