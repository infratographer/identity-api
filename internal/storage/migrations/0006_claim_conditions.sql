-- +goose Up
ALTER TABLE issuers
ADD COLUMN conditions VARCHAR;
-- +goose Down
ALTER TABLE issuers DROP COLUMN conditions;
