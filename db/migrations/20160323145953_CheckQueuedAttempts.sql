-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE queued_jobs ADD CONSTRAINT queued_jobs_attempts_check CHECK(attempts >= 0);


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE queued_jobs DROP CONSTRAINT queued_jobs_attempts_check;
