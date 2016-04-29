-- +goose Up
ALTER TABLE jobs ADD CONSTRAINT "jobs_attempts_check" CHECK(attempts > 0);
ALTER TABLE jobs ADD CONSTRAINT "jobs_concurrency_check" CHECK(concurrency > 0);


-- +goose Down
ALTER TABLE jobs DROP CONSTRAINT "jobs_attempts_check";
ALTER TABLE jobs DROP CONSTRAINT "jobs_concurrency_check";
