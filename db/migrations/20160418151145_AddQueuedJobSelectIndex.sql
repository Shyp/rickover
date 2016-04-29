-- +goose Up
CREATE INDEX find_queued_job on queued_jobs(name, run_after) WHERE status='queued';

-- +goose Down
DROP INDEX find_queued_job;
