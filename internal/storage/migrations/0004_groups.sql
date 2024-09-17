-- +goose Up
CREATE TABLE groups (
  id STRING PRIMARY KEY NOT NULL,
  owner_id STRING NOT NULL,
  name STRING NOT NULL,
  description STRING NOT NULL DEFAULT '',
  UNIQUE(owner_id, name)
);

-- +goose Down
DROP TABLE groups;
