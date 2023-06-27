-- +goose Up
CREATE TABLE issuers (
    id        VARCHAR(29) PRIMARY KEY NOT NULL,
    owner_id VARCHAR(29) NOT NULL,
    uri       STRING NOT NULL UNIQUE,
    name      STRING NOT NULL,
    jwksuri   STRING NOT NULL,
    mappings  STRING
);

CREATE TABLE user_info (
    id     VARCHAR(29) PRIMARY KEY NOT NULL,
    name   STRING NOT NULL,
    email  STRING NOT NULL,
    sub    STRING NOT NULL,
    iss_id VARCHAR(29) NOT NULL REFERENCES issuers(id),
    UNIQUE (iss_id, sub)
);
