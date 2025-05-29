-- +goose Up
CREATE TABLE groups (
  id VARCHAR PRIMARY KEY NOT NULL,
  owner_id VARCHAR NOT NULL,
  name VARCHAR NOT NULL,
  description VARCHAR NOT NULL DEFAULT '',
  UNIQUE(owner_id, name)
);
-- +goose Down
DROP TABLE groups;
