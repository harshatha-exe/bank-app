CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(30) UNIQUE NOT NULL CHECK (name IN ('CUSTOMER', 'OPERATIONS', 'ADMIN')),
    description TEXT
);

CREATE INDEX idx_roles_name ON roles (name);