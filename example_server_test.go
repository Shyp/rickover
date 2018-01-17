// Run the rickover server.
//
// All of the project defaults are used. There is one authenticated user for
// basic auth, the user is "test" and the password is "hymanrickover". You will
// want to copy this binary and add your own authentication scheme.
package rickover

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Shyp/go-simple-metrics"
	"github.com/Shyp/rickover/config"
	"github.com/Shyp/rickover/models/db"
	"github.com/Shyp/rickover/server"
	"github.com/Shyp/rickover/setup"
	"github.com/gorilla/handlers"
)

var serverDbConns int

func init() {
	var err error
	serverDbConns, err = config.GetInt("PG_SERVER_POOL_SIZE")
	if err != nil {
		log.Printf("Error getting database pool size: %s. Defaulting to 10", err)
		serverDbConns = 10
	}

	metrics.Namespace = "rickover.server"

	// Change this user to a private value
	server.AddUser("test", "hymanrickover")
}

func Example_server() {
	if err := setup.DB(db.DefaultConnection, serverDbConns); err != nil {
		log.Fatal(err)
	}

	metrics.Start("web")

	go setup.MeasureActiveQueries(5 * time.Second)

	log.Println("Listening on port 9090")
	log.Fatal(http.ListenAndServe(":9090", handlers.LoggingHandler(os.Stdout, server.DefaultServer)))
}
