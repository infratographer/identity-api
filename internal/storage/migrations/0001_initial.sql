-- +goose Up
CREATE TABLE issuers (
    id        UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    uri       STRING NOT NULL UNIQUE,
    name      STRING NOT NULL,
    jwksuri   STRING NOT NULL,
    mappings  STRING
);

CREATE TABLE user_info (
    id     UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
    name   STRING NOT NULL,
    email  STRING NOT NULL,
    sub    STRING NOT NULL,
    iss_id UUID NOT NULL REFERENCES issuers(id),
    UNIQUE (iss_id, sub)
);
