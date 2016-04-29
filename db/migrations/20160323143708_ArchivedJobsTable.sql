-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TYPE archived_job_status AS enum('succeeded', 'failed', 'expired');
CREATE TABLE archived_jobs (
	id UUID PRIMARY KEY,
	name TEXT NOT NULL REFERENCES jobs(name),
	attempts SMALLINT NOT NULL CHECK (attempts >= 0),
	status archived_job_status NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
	data JSONB NOT NULL
);


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE archived_jobs;
DROP TYPE archived_job_status;
