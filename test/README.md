# Test class

If a test involves integrating with the filesystem, the network, or the
database, it should go here. The converse is no tests that do those things
should go anywhere else.

Tests in this folder are run one at a time, unless you specify `t.Parallel()`,
which you can do if the test doesn't successfully write any state that could be
read by any other test. HTTP requests can be made in parallel.

## blank.go

Unfortunately godep can't build packages that contain *only* tests, hence the
use of the `blank.go` file to trick Go.
