-- +goose Up
CREATE TABLE group_members (
  group_id VARCHAR NOT NULL REFERENCES groups(id),
  subject_id VARCHAR NOT NULL,
  primary key (group_id, subject_id)
);
CREATE INDEX IF NOT EXISTS group_memberships_subject_id_index ON group_members (subject_id);
-- +goose Down
DROP INDEX group_memberships_subject_id_index;
DROP TABLE group_members;
