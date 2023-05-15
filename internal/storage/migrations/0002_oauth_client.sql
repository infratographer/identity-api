-- +goose Up
CREATE TABLE oauth_clients (
    id        VARCHAR(29) PRIMARY KEY NOT NULL,
    tenant_id VARCHAR(29) NOT NULL,
    name      STRING NOT NULL,
    secret    STRING NOT NULL,
    audience  STRING NOT NULL
);
