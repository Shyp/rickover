package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/Shyp/rickover/models/db"
	"github.com/Shyp/rickover/setup"
)

func SetUp(t testing.TB) {
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "postgres://rickover@localhost:5432/rickover_test?sslmode=disable&timezone=UTC")
	}
	if err := setup.DB(db.DefaultConnection, 10); err != nil {
		t.Fatal(err)
	}
}

// TruncateTables deletes all records from the database.
func TruncateTables() error {
	getTableDelete := func(table string) string {
		return fmt.Sprintf("DELETE FROM %[1]s", table)
	}
	_, err := db.Conn.Exec(fmt.Sprintf("BEGIN; %s;\n%s;\n%s; COMMIT",
		getTableDelete("archived_jobs"),
		getTableDelete("queued_jobs"),
		getTableDelete("jobs"),
	))
	return err
}

// TearDown deletes all records from the database, and marks the test as failed
// if this was unsuccessful.
func TearDown(t testing.TB) {
	if db.Connected() {
		if err := TruncateTables(); err != nil {
			t.Fatal(err)
		}
	}
}
