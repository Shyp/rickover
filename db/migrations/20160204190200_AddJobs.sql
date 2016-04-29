-- +goose Up
CREATE TYPE delivery_strategy AS enum('at_most_once', 'at_least_once');
CREATE TABLE jobs (
  name text PRIMARY KEY,
  delivery_strategy delivery_strategy NOT NULL,
  attempts SMALLINT NOT NULL,
  concurrency SMALLINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);


-- +goose Down
DROP TABLE jobs;
DROP TYPE delivery_strategy;
