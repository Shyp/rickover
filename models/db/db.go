// Definitions of database objects, and logic for connecting to the database.
package db

import (
	"database/sql"
	"sync"
)

var mu sync.Mutex

// Conn is a shared connection used by all database queries.
var Conn *sql.DB

// Connector establishes a connection to a Postgres database, with the given
// number of connections, and stores the connection in conn.
type Connector interface {
	Connect(conn *sql.DB, dbConns int) error
}

// Connected returns true if a connection exists to the database.
func Connected() bool {
	mu.Lock()
	defer mu.Unlock()
	return Conn != nil
}
