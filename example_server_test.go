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

func Example_server() {
	dbConns, err := config.GetInt("PG_SERVER_POOL_SIZE")
	if err != nil {
		log.Printf("Error getting database pool size: %s. Defaulting to 10", err)
		dbConns = 10
	}

	if err = setup.DB(db.DefaultConnection, dbConns); err != nil {
		log.Fatal(err)
	}

	metrics.Namespace = "rickover.server"
	metrics.Start("web")

	go setup.MeasureActiveQueries(5 * time.Second)

	// Change this user to a private value
	server.AddUser("test", "hymanrickover")
	s := server.Get(server.DefaultAuthorizer)

	log.Println("Listening on port 9090\n")
	log.Fatal(http.ListenAndServe(":9090", handlers.LoggingHandler(os.Stdout, s)))
}
