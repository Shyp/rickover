// Package db contains logic for connecting to the database.
package db

import (
	"database/sql"
	"errors"
	"os"
	"sync"
)

// DefaultConnection connects to a Postgres database using the DATABASE_URL
// environment variable.
var DefaultConnection = &DatabaseURLConnector{}

// DatabaseURLConnector connects to the database using the DATABASE_URL
// environment variable.
type DatabaseURLConnector struct {
	mu sync.Mutex
}

// Connect to the database using the DATABASE_URL environment variable with the
// given number of database connections, and store the result in conn.
func (dc *DatabaseURLConnector) Connect(conn *sql.DB, dbConns int) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if conn == nil {
		return errors.New("setup: Cannot assign to nil conn")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("setup: No value provided for DATABASE_URL, cannot connect")
	}
	d, err := sql.Open("postgres", url)
	if err != nil {
		return err
	}
	d.SetMaxOpenConns(dbConns)
	if dbConns > 100 {
		d.SetMaxIdleConns(dbConns - 20)
	} else if dbConns > 50 {
		d.SetMaxIdleConns(dbConns - 10)
	} else if dbConns > 10 {
		d.SetMaxIdleConns(dbConns - 3)
	} else if dbConns > 5 {
		d.SetMaxIdleConns(dbConns - 2)
	}
	*conn = *d
	return nil
}

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
