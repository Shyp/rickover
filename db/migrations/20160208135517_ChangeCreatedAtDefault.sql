-- +goose Up
ALTER TABLE jobs ALTER COLUMN created_at SET DEFAULT now();


-- +goose Down
-- This is incorrect. now() at time zone 'utc' is a timestamp without time
-- zone, so clients not using UTC timezone will append their own timezone to
-- it.
ALTER TABLE jobs ALTER COLUMN created_at SET DEFAULT now() at time zone 'utc';
