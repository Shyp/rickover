-- +goose Up
CREATE INDEX queued_jobs_created_at ON queued_jobs(created_at ASC);

-- +goose Down
DROP INDEX queued_job_created_at;
