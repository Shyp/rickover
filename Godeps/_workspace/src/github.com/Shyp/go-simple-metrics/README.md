# go-metrics

This is a stupid simple wrapper around the Librato go library. There are only
three functions, the ones we use the most often.

## Example Usage

```go
func init() {
    metrics.Namespace = "api"
    source := "prod.web.1"
    metrics.Start(source)
}

metrics.Increment("signup")
start := time.Now()
time.Sleep(3*time.Second)
metrics.Time("long_op", time.Since(start))

metrics.Measure("queue_depth", 57)
```

## Configuration

Set `LIBRATO_TOKEN` in your environment; this gets passed to the librato client
and will publish metrics every ten seconds.

## Debugging

Set DEBUG=metrics in your environment to print all metrics to stdout in
realtime.
