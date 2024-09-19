-- +goose Up
CREATE TABLE group_members (
  group_id STRING NOT NULL REFERENCES groups(id),
  subject_id STRING NOT NULL,

  index group_memberships_subject_id_index (subject_id),
  primary key (group_id, subject_id)
);

-- +goose Down
DROP TABLE group_members;
