# Rickover

This holds the code for a scheduler and a job queue written in Go and
backed by Postgres.

The goals/features of this project are:

- Visibility into the system using a tool our team is familiar with (Postgres)
- Correctness - jobs shouldn't get stuck, or dequeued twice, unless that's
  desirable
- Good memory performance - with 300 dequeuers, the server and worker take
  about 30MB in total.
- No long-running database queries, or open transactions
- All queue actions have an HTTP API.
- All enqueue/dequeue actions are idempotent - it's OK if any part of the
  system gets restarted

It might not be the most performant, but it should be easy to use and deploy!

## Server Endpoints

The only supported content type for uploads and responses is JSON.

#### Create a job type

To start enqueueing and dequeueing work, you need to create a job type. Define
a job type with a name, a delivery strategy (idempotent == `"at_least_once"`,
not idempotent == `"at_most_once"`), and a concurrency - the maximum number
of jobs that can be in flight at once. If the job is idempotent, you can add
"attempts" - the number of times to try to send the job to the downstream
server before giving up.

```
POST /v1/jobs
{
    "id": "invoice-shipments",
    "delivery_strategy": "at_least_once",
    "attempts": 3,
    "concurrency": 5
}
```

This returns a [models.Job][job-type] on success.

[job-type]: https://godoc.org/github.com/Shyp/rickover/models#Job

#### Enqueue a new job

Once you have a job type, you can enqueue new jobs. Note the client is
responsible for generating a UUID.

```
PUT /v1/jobs/invoice-shipments/job_282227eb-3c76-4ef7-af7e-25dff933077f
{
    "data": {
        "shipmentId": "shp_123",
    }
    "id": "job_282227eb-3c76-4ef7-af7e-25dff933077f",
    "run_after": "2016-01-11T18:26:26.000Z",
    "expires_at": "2016-01-11T20:26:26.000Z"
}
```

This inserts a record into the `queued_jobs` table and returns a
[models.QueuedJob][queued-job]. The client can and should retry in the event of
failure.

You can put any valid JSON in the `data` field; we'll send this to the
downstream worker.

There are two special fields - `run_after` indicates the earliest possible
time this job can run (or `null` to indicate it can run now), and `expires_at`
indicates the latest possible time this job can run. If a job is dequeued after
the `expires_at` date, we don't send it to the downstream worker, and insert it
immediately into the `archived_jobs` table with status `expired`.

[queued-job]: https://godoc.org/github.com/Shyp/rickover/models#QueuedJob

#### Record a job's success or failure

Once the downstream worker has completed work, record the status of the job by
making a POST request to the same URI.

```
POST /v1/jobs/invoice-shipments/job_123 HTTP/1.1
{
    "status": "succeeded"
    "attempt": 3,
}
```

Note you must include the attempt number in your callback; we use this
for idempotency, and to avoid stale writes. Valid values for `status` are
"succeeded" or "failed". If a failed job is retryable, we'll insert the job
back into the `queued_jobs` table with a `run_after` date set a small amount
of time in the future. If a failed job is retryable but should not be retried,
include `"retryable": false` in the body of the POST request, which will
immediately archive the job.

#### Replay a job

This is handy if the initial job failed, the downstream server had an outage,
or you want to re-run something on an adhoc basis.

```
POST /v1/jobs/invoice-shipments/job_123/replay HTTP/1.1
```

Will create a new UUID and enqueue the job to be run immediately. If you
attempt to replay an expired job, the new job will be immediately archived with
a status of "expired".

#### Get information about a job

```
GET /v1/jobs/invoice-shipments/job_123 HTTP/1.1
```

This looks in the queued_jobs table first, then the archived_jobs table, and
returns whatever it finds. Note the fields in these tables don't match up 100%.


### Server Authentication

By default, the server uses an in-memory secret for authentication. Call
[`server.AddUser`][add-user] to add an authenticated user and password for the
[DefaultServer][default-server].

You can use your own authentication scheme with any code that satisifies the
[server.Authorizer][authorizer] interface:

```go
// Authorizer can authorize the given user and token to access the API.
type Authorizer interface {
	Authorize(user string, token string) error
}
```

Then, get a http.Handler with your authorizer by calling

```go
import "github.com/Shyp/rickover/server"

handler := server.Get(authorizer)
http.ListenAndServe(":9090", handler)
```

[add-user]: https://godoc.org/github.com/Shyp/rickover/server#AddUser
[default-server]: https://godoc.org/github.com/Shyp/rickover/server#pkg-variables
[authorizer]: https://godoc.org/github.com/Shyp/rickover/server#Authorizer

## Processing jobs

When you get a job from the database, you can do whatever you want with it -
your dequeuer just needs to satisfy the [Worker][worker] interface.

```go
// A Worker does some work with a QueuedJob.
type Worker interface {
	DoWork(*models.QueuedJob) error
}
```

[worker]: https://godoc.org/github.com/Shyp/rickover/dequeuer#Worker

A default Worker is provided as [services.JobProcessor][job-processor],
which makes an API request to a downstream service. The default client is
[downstream.Client][downstream-client]. You'll need to set the URL
and password for the downstream service:

```go
import "github.com/Shyp/rickover/dequeuer"
import "github.com/Shyp/rickover/services"

func main() {
	password := "hymanrickover"
	// Basic auth - username "jobs", password password
	jp := services.NewJobProcessor("http://downstream-service.example.com", password)

	pools, err := dequeuer.CreatePools(jp)
	fmt.Println(err)
}
```

[job-processor]: https://godoc.org/github.com/Shyp/rickover/services#JobProcessor
[downstream-client]: https://godoc.org/github.com/Shyp/rickover/downstream#Client

The [downstream.Client][downstream-client] will make a POST request to
`/v1/jobs/:job-name/:job-id`:

```
POST /v1/jobs/invoice-shipment/job_123 HTTP/1.1
Host: downstream.shyp.com
Content-Type: application/json
Accept: application/json
{
    "data": {
        "shipmentId": "shp_123"
    },
    "id": "job_123",
    "attempts": 3
}
```

## Callbacks

All actions in the system are designed to be short-lived. When the downstream
server has finished processing the job, it should make a callback to the
Rickover server, reporting on the status of the job, with `status` set to
`succeeded` or `failed`.

```
POST /v1/jobs/invoice-shipments/job_123 HTTP/1.1
Host: rickover.shyp.com
Content-Type: application/json
{
    "status": "succeeded"
    "attempt": 3,
}
```

If this request times out or errors, you can try it again; the `attempt` number
is used to avoid making a stale update.

You can also report status of a job by calling
[services.HandleStatusCallback][status-callback] directly, with success or
failure.

[status-callback]: https://godoc.org/github.com/Shyp/rickover/services#HandleStatusCallback

## Failure Handling

If the downstream worker never hits the callback, the JobProcessor will time
out after 5 minutes and mark the job as failed.

If the dequeuer gets killed while waiting for a response, we'll time out the
job after 7 minutes, and mark it as failed. (This means the maximum allowable
time for a job is 7 minutes.)

## Dashboard

The homepage can embed an iframe of your choice, configurable via the
`HOMEPAGE_IFRAME_URL` environment variable. We set up a Librato space with the
metrics we send from this service, and embed that in the homepage:

<img src="https://monosnap.com/file/nr399cxUwpBkEk5gAjJ5hTG4c3aR9J.png">

#### Stuck jobs

If a dequeuer gets restarted after it's Acquire()d a job but before it can send
it downstream, or if the downstream worker gets restarted before it can hit the
callback, the job can get stuck in-progress indefinitely. Run WatchStuckJobs in
a goroutine to periodically check for in-progress jobs and mark them as failed:

```go
// This should be longer than the timeout in the JobProcessor
stuckJobTimeout := 7 * time.Minute
go services.WatchStuckJobs(1*time.Minute, stuckJobTimeout)
```

## Database Table Layout

There are three tables, plus one for keeping track of ran migrations.

- `jobs` - Contains information about a job's name, retry strategy, desired
  concurrency.

```
                          Table "public.jobs"
      Column       |           Type           |       Modifiers
-------------------+--------------------------+------------------------
 name              | text                     | not null
 delivery_strategy | delivery_strategy        | not null
 attempts          | smallint                 | not null
 concurrency       | smallint                 | not null
 created_at        | timestamp with time zone | not null default now()
Indexes:
    "jobs_pkey" PRIMARY KEY, btree (name)
Check constraints:
    "jobs_attempts_check" CHECK (attempts > 0)
    "jobs_concurrency_check" CHECK (concurrency >= 0)
Referenced by:
    TABLE "archived_jobs" CONSTRAINT "archived_jobs_name_fkey" FOREIGN KEY (name) REFERENCES jobs(name)
    TABLE "queued_jobs" CONSTRAINT "queued_jobs_name_fkey" FOREIGN KEY (name) REFERENCES jobs(name)
```

- `queued_jobs` - The "hot" table, this contains rows that are scheduled to be
dequeued. Should be small, so queries are fast.

```
                   Table "public.queued_jobs"
   Column   |           Type           |       Modifiers
------------+--------------------------+------------------------
 id         | uuid                     | not null
 name       | text                     | not null
 attempts   | smallint                 | not null
 run_after  | timestamp with time zone | not null
 expires_at | timestamp with time zone |
 created_at | timestamp with time zone | not null default now()
 updated_at | timestamp with time zone | not null default now()
 status     | job_status               | not null
 data       | jsonb                    | not null
Indexes:
    "queued_jobs_pkey" PRIMARY KEY, btree (id)
    "find_queued_job" btree (name, run_after) WHERE status = 'queued'::job_status
    "queued_jobs_created_at" btree (created_at)
Check constraints:
    "queued_jobs_attempts_check" CHECK (attempts >= 0)
Foreign-key constraints:
    "queued_jobs_name_fkey" FOREIGN KEY (name) REFERENCES jobs(name)
```

- `archived_jobs` - Insert-only table containing historical records of all
jobs. May grow very large.

```
            Table "public.archived_jobs"
   Column   |           Type           |       Modifiers
------------+--------------------------+------------------------
 id         | uuid                     | not null
 name       | text                     | not null
 attempts   | smallint                 | not null
 status     | archived_job_status      | not null
 created_at | timestamp with time zone | not null default now()
 data       | jsonb                    | not null
Indexes:
    "archived_jobs_pkey" PRIMARY KEY, btree (id)
Check constraints:
    "archived_jobs_attempts_check" CHECK (attempts >= 0)
Foreign-key constraints:
    "archived_jobs_name_fkey" FOREIGN KEY (name) REFERENCES jobs(name)
```

## Example servers and dequeuers

Example server and dequeuer instances are stored in commands/server and
commands/dequeuer. You will probably want to modify these to provide your own
authentication scheme.

## Configure the server

You can use the following variables to tune the server:

- `PG_SERVER_POOL_SIZE` - Maximum number of database connections from an
individual instance. Across every database connection in the cluster, you want
to have the number of active Postgres connections equal to 2 * (num CPUs on the
Postgres machine). Currently set to 15.

- `PORT` - which port to listen on.

- `LIBRATO_TOKEN` - This library uses Librato for metrics. This environment
variable sets the Librato token for publishing.

- `DATABASE_URL` - Postgres database URL. Currently only connections to the
  primary are allowed, there are not a lot of reads in the system, and all
  queries are designed to be short.

## Configure the dequeuer

The number of dequeuers is determined by the number of entries in the `jobs`
table. There is currently no way to adjust the number of dequeuers on the fly,
you must update the database and then restart the worker process.

- `PG_WORKER_POOL_SIZE` - How many workers to use. Workers hit Postgres in a
busy loop asking for work with a `SELECT ... FOR UPDATE`, which skips rows
if they are active, so queries from the worker tend to cause more active
connections than those from the server.

- `DATABASE_URL` - Postgres database URL. Currently only connections to the
  primary are allowed, there are not a lot of reads in the system, and all
  queries are designed to be short.

- `DOWNSTREAM_URL` - When you dequeue a job, hit this URL to tell something to
  do some work.

- `DOWNSTREAM_WORKER_AUTH` - Basic auth password for the downstream service
  (user is "jobs").

## Local development

We use [goose][goose] for database migrations. The test database is
`rickover_test` and the development database is `rickover`. The authenticating
user is `rickover`.

To run all migrations, run:

```
goose --env=test up
```

To get the database status:

```
goose --env=test status
```

You should also be able to use goose to run migrations in your production
environment. Set the DATABASE_URL environment variable to a Postgres string,
then use the `cluster` environment, for example

```
DATABASE_URL=$(heroku config:get DATABASE_URL --app myapp) goose --env=cluster up
```

[goose]: https://bitbucket.org/liamstask/goose

### Start the server

```
make serve
```

Will start the example server on port 8080.

### Start the dequeuer

```
make dequeue
```

Will try to pull jobs out of the database and send them to the downstream
worker. Note you will need to set `DOWNSTREAM_WORKER_AUTH` as the basic auth
password for the downstream service (the user is hardcoded to "jobs"), and
`DOWNSTREAM_URL` as the URL to hit when you have a job to dequeue.

## Debugging variables

- `DEBUG_HTTP_TRAFFIC` - Dump all incoming and outgoing http traffic to stdout

## Run the tests

```
make test
```

The race detector takes longer to run, so we only enable it in CircleCI, but
you can run tests with the race detector enabled:

```
make race-test
```

## View the docs

Run `make docs`, which will start a docs server on port 6060 and open it to the
right documentation page. All public API's will be present, and most function
calls should be documented - some with examples.

## Working with Godep

Godep is the tool for bringing all dependencies into the project. It's
unfortunately a pain in the neck. The basic workflow you want is:

```
# Rewrite all import paths to not have the word Godeps in them
godep save -r=false ./...

# Rewrite all import paths to refer to Godep equivalents
godep save -r ./...

# Update a dependency. First go to the package locally, and point it at the new
# version via "git checkout master" or whatever. Then in this repo, run:
godep update github.com/Shyp/goshyp/...
```

If you're only adding new code, you'll likely only need to run `godep save -r
./...`. The good news is that the CircleCI tests will fail if you screw this
up, so you can't really deploy unless it works.

## Benchmarks

Running locally on my 2014 MBP, I was able to dequeue 50,000 jobs per minute.
Note the downstream Node server is too slow, even when run on 4 cores. You will
want to start the `downstream-server` in this project instead.

Some benchmark numbers are here: https://docs.google.com/a/shyp.co/spreadsheets/d/1KF3pqCczDMRXZcq-ZqpQeGKo4sPclThltWhsxHdUPTc/edit?usp=sharing

The second bottleneck was the database. Note the database performed best when
the numbers of connection counts and dequeuers were low. In the cluster we will
want to have a higher number of dequeuers, simply because we aren't enqueueing
as many jobs, it's more important to be fast when we need speed than to worry
about the optimal number for peak performance.

In the cluster I was able to dequeue 7,000 jobs per minute with a single $25
web node, a $25 dequeuer node, a $50 database and a $25 Node.js worker. The
first place I would look to improve this would be to increase the number of
Node (downstream) dynos.

I used [`boom`][boom] for enqueuing jobs; I turned off the dequeuer, enqueued
30000 jobs, then started the dequeuer and measured the timestamp difference
between the first enqueued job and the last.

There's a builtin `random_id` endpoint which will generate a UUID for you, for
doing load testing.

```
boom -n 30000 -c 100 -d '{"data": {"user-agent": "boom"}}' -m PUT http://localhost:9090/v1/jobs/echo/random_id
```

[boom]: https://github.com/rakyll/boom

## Supported versions

The database uses `jsonb`, which is only available in Postgres 9.4 and beyond.
The Go server exposes `http/pprof/trace`, which is only available in Go 1.5 and
beyond.

You can probably fork the project to remove the http/pprof handlers and replace
jsonb with json and it should compile/run fine.

## Single points of failure

This project has two points of failure:

- If the database goes down, you can't enqueue any new jobs, or process
any work. This is acceptable for most companies and projects. If this is
unacceptable you may want to check out a distributed scheduler like Chronos, or
use something else for the job queue, like SQS.

- If the dequeuer goes down, you can't process any work. You can run multiple
dequeuers, or manually split the single concurrency value across multiple
machines, or just wait until the machine comes back up again. Note if the
dequeuer goes down you can still enqueue jobs, the database queue will continue
to grow.

## Suggestions for scaling the project

- Pull jobs out of the database in batches, and send them to the dequeuers over
channels.

- Use it only as a scheduler, and move the job queue to SQS or something else.

- Run the server on multiple machines. The worker can't run on multiple
  machines without violating the concurrency guarantees.

- Run the worker on multiple machines, and ignore or update the concurrency
  guarantees.

- Run the downstream worker on a larger number of machines.

- Shard the Postgres database so jobs A-M are in one database, and N-Z are in
another. Would need to update `db.Conn` to be an interface, or wrap it behind a
function.

- Delete/archive all rows from `archived_jobs` that are older than 180 days, on
a rolling basis. I doubt this will help much, but it might.

- Get a bigger Postgres database.

- Upgrade to Postgres 9.5 and update the Acquire() strategy to use SKIP LOCKED.

## Roadmap

- Allow UPDATEs to jobs once they've been created

- API for retrieving recent jobs/paging through archived jobs, by name

- Dequeuer listens on a local socket/port, so you can send commands to it to
  add/remove dequeuers on the fly.
