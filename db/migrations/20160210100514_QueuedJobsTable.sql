-- +goose Up
CREATE TYPE job_status AS enum('queued', 'in-progress');
CREATE TABLE queued_jobs (
	id UUID PRIMARY KEY,
	name TEXT NOT NULL REFERENCES jobs(name),
	attempts SMALLINT NOT NULL,
	run_after TIMESTAMP WITH TIME ZONE NOT NULL,
	expires_at TIMESTAMP WITH TIME ZONE,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
	updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
	status job_status NOT NULL,
	data JSONB NOT NULL
);

-- +goose Down
DROP TABLE queued_jobs;
DROP TYPE job_status;
