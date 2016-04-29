The server is a cheap replica of the Node downstream worker. When I tried to
benchmark the job queue/scheduler, Node was the bottleneck. Even a bare Express
app like this:

```javascript
var express = require('express');

const UNAUTHORIZED = {
  title: 'Please provide a valid username and password',
  detail: 'You did not provide credentials, or the provided credentials are not correct',
  id: 'unauthorized'
};

var app = express();
app.post('/v1/jobs/echo/:id', function(req, res) {
  res.status(401).json(UNAUTHORIZED);
});
app.listen(6799);
```

takes about 50ms to respond to every request, with 200 concurrency.

Instead here's a Go server that responds to every incoming request with a 202
and then immediately fires the success callback in a goroutine.
