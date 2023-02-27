-- +goose Up
CREATE TABLE oauth_clients (
    id        UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name      STRING NOT NULL,
    secret    STRING NOT NULL,
    audience  STRING NOT NULL,
    scope     STRING NOT NULL
);
