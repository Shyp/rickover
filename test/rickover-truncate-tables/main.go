package main

import (
	"log"

	"github.com/Shyp/rickover/models/db"
	"github.com/Shyp/rickover/setup"
	"github.com/Shyp/rickover/test"
)

func main() {
	if err := setup.DB(db.DefaultConnection, 1); err != nil {
		log.Fatal(err)
	}
	if err := test.TruncateTables(nil); err != nil {
		log.Fatal(err)
	}
}
