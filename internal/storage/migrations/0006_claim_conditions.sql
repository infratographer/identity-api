-- +goose Up
ALTER TABLE issuers ADD COLUMN conditions STRING;
-- +goose Down
ALTER TABLE issuers DROP COLUMN conditions;
