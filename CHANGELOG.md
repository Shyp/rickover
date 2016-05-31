## Version 0.37

Fixes a crashing error in dequeuer.CreatePools (and adds tests so it can't
happen again)

## Version 0.36

The `job_` prefix is now retrieved as part of the database query instead of
being attached in the model. You'll need to update `go-types` to version 0.13
at the latest.

## Version 0.35

- Fixes an error in a test committed shortly before version 0.34.

## Version 0.34

- Support marking failed jobs as un-retryable; pass `{"status": "failed",
"retryable": false}` in your status callback endpoint to immediately archive
the job.

- The 0.33 git tag doesn't compile due to the error fixed here:
https://github.com/Shyp/bump_version/commit/2dc60a73949ae5e42468d475a90e76619dbc67a6.
Adds regression tests to ensure this doesn't happen again.

## Version 0.33

- All uses of `Id` have been renamed to `ID`, per the Go Code Review Comments
guidelines. I don't like breaking this, but I'd rather keep the naming
idiomatic, Go will detect incorrect references at compile time, and I haven't
received any evidence that anyone else is using the project, so I am not too
worried about breaking compatibility in the wild.

- When replaying a job, use the `expires_at` value from the old job to
  determine whether to re-run it.

- Enabled several skipped tests and improved their speed/reduced their size.

## Version 0.32

- The `archived_jobs` table now has an `expires_at` column storing when the job
expired, or should have expired. This is useful for replaying jobs - you can
batch replay jobs and the server will correctly mark them as expired.
