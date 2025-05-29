-- +goose Up
CREATE TABLE oauth_clients (
    id VARCHAR(29) PRIMARY KEY NOT NULL,
    owner_id VARCHAR(29) NOT NULL,
    name VARCHAR NOT NULL,
    secret VARCHAR NOT NULL,
    audience VARCHAR NOT NULL
);
