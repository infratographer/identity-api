-- +goose Up
CREATE TABLE issuers (
    id VARCHAR(29) PRIMARY KEY NOT NULL,
    owner_id VARCHAR(29) NOT NULL,
    uri VARCHAR NOT NULL UNIQUE,
    name VARCHAR NOT NULL,
    jwksuri VARCHAR NOT NULL,
    mappings VARCHAR
);
CREATE TABLE user_info (
    id VARCHAR(29) PRIMARY KEY NOT NULL,
    name VARCHAR NOT NULL,
    email VARCHAR NOT NULL,
    sub VARCHAR NOT NULL,
    iss_id VARCHAR(29) NOT NULL REFERENCES issuers(id),
    UNIQUE (iss_id, sub)
);
