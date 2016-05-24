package db

import (
	"fmt"
	"os"
	"testing"

	"github.com/Shyp/rickover/models/db"
	"github.com/Shyp/rickover/setup"
)

func SetUp(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "postgres://rickover@localhost:5432/rickover_test?sslmode=disable&timezone=UTC")
	}
	if err := setup.DB(db.DefaultConnection, 10); err != nil {
		t.Fatal(err)
	}
}

func TearDown(t *testing.T) {
	getTableDelete := func(table string) string {
		return fmt.Sprintf("DELETE FROM %[1]s", table)
	}
	if db.Connected() {
		_, err := db.Conn.Exec(fmt.Sprintf("BEGIN; %s;\n%s;\n%s; COMMIT",
			getTableDelete("archived_jobs"),
			getTableDelete("queued_jobs"),
			getTableDelete("jobs"),
		))
		if err != nil {
			t.Fatal(err)
		}
	}
}
