package dberror

import (
	"database/sql"
	"os"
	"sync"
	"testing"

	"github.com/letsencrypt/boulder/test"
	"github.com/lib/pq"
)

const uuid = "3c7d2b4a-3fc8-4782-a518-4ce9efef51e7"
const uuid2 = "91f47e99-d616-4d8c-9c02-cbd13bceac60"
const email = "test@example.com"
const email2 = "test2@example.com"

var db *sql.DB

var mu sync.Mutex

func setUp(t *testing.T) {
	mu.Lock()
	if db != nil {
		mu.Unlock()
		return
	}
	ci := os.Getenv("CI")
	var err error
	if ci == "" {
		db, err = sql.Open("postgres", "postgres://localhost/dberror?sslmode=disable")
	} else {
		db, err = sql.Open("postgres", "postgres://ubuntu@localhost/circle_test?sslmode=disable")
	}
	mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
}

func tearDown(t *testing.T) {
	if db != nil {
		_, err := db.Exec("DELETE FROM accounts CASCADE; DELETE FROM payments CASCADE;")
		test.AssertNotError(t, err, "")
	}
}

func TestNilError(t *testing.T) {
	test.AssertEquals(t, GetError(nil), nil)
}

func TestNotNull(t *testing.T) {
	t.Parallel()
	setUp(t)
	_, err := db.Exec("INSERT INTO accounts (id) VALUES (null)")
	dberr := GetError(err)
	switch e := dberr.(type) {
	case *Error:
		test.AssertEquals(t, e.Error(), "No id was provided. Please provide a id")
		test.AssertEquals(t, e.Column, "id")
		test.AssertEquals(t, e.Table, "accounts")
	default:
		t.Fail()
	}
}

func TestDefaultConstraint(t *testing.T) {
	// this test needs to go before the Register() below... not great, add an
	// unregister or clear out the map or something
	setUp(t)
	_, err := db.Exec("INSERT INTO accounts (id, email, balance) VALUES ($1, $2, -1)", uuid, email)
	dberr := GetError(err)
	switch e := dberr.(type) {
	case *Error:
		test.AssertEquals(t, e.Error(), "new row for relation \"accounts\" violates check constraint \"accounts_balance_check\"")
		test.AssertEquals(t, e.Table, "accounts")
	default:
		t.Fail()
	}
}

func TestCustomConstraint(t *testing.T) {
	setUp(t)
	defer func() {
		constraintMap = map[string]*Constraint{}
	}()
	constraint := &Constraint{
		Name: "accounts_balance_check",
		GetError: func(e *pq.Error) *Error {
			return &Error{
				Message:  "Cannot write a negative balance",
				Severity: e.Severity,
				Table:    e.Table,
				Detail:   e.Detail,
				Code:     string(e.Code),
			}
		},
	}
	RegisterConstraint(constraint)
	_, err := db.Exec("INSERT INTO accounts (id, email, balance) VALUES ($1, $2, -1)", uuid, email)
	dberr := GetError(err)
	switch e := dberr.(type) {
	case *Error:
		test.AssertEquals(t, e.Error(), "Cannot write a negative balance")
		test.AssertEquals(t, e.Table, "accounts")
	default:
		t.Fail()
	}
}

func TestInvalidUUID(t *testing.T) {
	t.Parallel()
	setUp(t)
	_, err := db.Exec("INSERT INTO accounts (id) VALUES ('foo')")
	dberr := GetError(err)
	switch e := dberr.(type) {
	case *Error:
		test.AssertEquals(t, e.Error(), "Invalid input syntax for type uuid: \"foo\"")
	default:
		t.Fail()
	}
}

func TestInvalidJSON(t *testing.T) {
	t.Parallel()
	setUp(t)
	_, err := db.Exec("INSERT INTO accounts (data) VALUES ('')")
	dberr := GetError(err)
	switch e := dberr.(type) {
	case *Error:
		test.AssertEquals(t, e.Error(), "Invalid input syntax for type json")
	default:
		t.Fail()
	}
}

func TestInvalidEnum(t *testing.T) {
	t.Parallel()
	setUp(t)
	_, err := db.Exec("INSERT INTO accounts (id, email, balance, status) VALUES ($1, $2, 1, 'blah')", uuid, email)
	dberr := GetError(err)
	switch e := dberr.(type) {
	case *Error:
		test.AssertEquals(t, e.Error(), "Invalid account_status: \"blah\"")
	default:
		t.Fail()
	}
}

func TestTooLargeInt(t *testing.T) {
	t.Parallel()
	setUp(t)
	_, err := db.Exec("INSERT INTO accounts (id, email, balance) VALUES ($1, $2, 40000)", uuid, email)
	dberr := GetError(err)
	switch e := dberr.(type) {
	case *Error:
		test.AssertEquals(t, e.Error(), "Smallint too large or too small")
	default:
		t.Fail()
	}
}

func TestUniqueConstraint(t *testing.T) {
	setUp(t)
	defer tearDown(t)
	query := "INSERT INTO accounts (id, email, balance) VALUES ($1, $2, 1)"
	_, err := db.Exec(query, uuid, email)
	test.AssertNotError(t, err, "")
	_, err = db.Exec(query, uuid, email)
	dberr := GetError(err)
	switch e := dberr.(type) {
	case *Error:
		test.AssertEquals(t, e.Error(), "A id already exists with this value (3c7d2b4a-3fc8-4782-a518-4ce9efef51e7)")
		test.AssertEquals(t, e.Column, "id")
		test.AssertEquals(t, e.Table, "accounts")
		test.AssertEquals(t, e.Code, CodeUniqueViolation)
	default:
		t.Fail()
	}
}

func TestUniqueFailureOnUpdate(t *testing.T) {
	setUp(t)
	defer tearDown(t)
	query := "INSERT INTO accounts (id, email, balance) VALUES ($1, $2, 1)"
	_, err := db.Exec(query, uuid, email)
	test.AssertNotError(t, err, "")
	_, err = db.Exec(query, uuid2, email2)
	test.AssertNotError(t, err, "")

	_, err = db.Exec("UPDATE accounts SET email = $1 WHERE id = $2", email, uuid2)
	dberr := GetError(err)
	switch e := dberr.(type) {
	case *Error:
		test.AssertEquals(t, e.Error(), "A email already exists with this value (test@example.com)")
		test.AssertEquals(t, e.Column, "email")
		test.AssertEquals(t, e.Table, "accounts")
		test.AssertEquals(t, e.Code, CodeUniqueViolation)
	default:
		t.Fail()
	}
}

func TestForeignKeyFailure(t *testing.T) {
	setUp(t)
	defer tearDown(t)
	query := "INSERT INTO payments (id, account_id) VALUES ($1, $2)"
	_, err := db.Exec(query, uuid, uuid2)

	dberr := GetError(err)
	switch e := dberr.(type) {
	case *Error:
		test.AssertEquals(t, e.Error(), "Can't save to payments because the account_id (91f47e99-d616-4d8c-9c02-cbd13bceac60) isn't present in the accounts table")
		test.AssertEquals(t, e.Column, "")
		test.AssertEquals(t, e.Table, "payments")
		test.AssertEquals(t, e.Code, CodeForeignKeyViolation)
	default:
		t.Fail()
	}
}

func TestCapitalize(t *testing.T) {
	t.Parallel()
	test.AssertEquals(t, capitalize("foo"), "Foo")
	test.AssertEquals(t, capitalize("foo bar baz"), "Foo bar baz")
}

func TestColumnFinder(t *testing.T) {
	t.Parallel()
	test.AssertEquals(t, findColumn("Key (id)=(blah) already exists."), "id")
	test.AssertEquals(t, findColumn("Key (foo bar)=(blah) already exists."), "foo bar")
	test.AssertEquals(t, findColumn("Unknown detail message"), "")
}

func TestValueFinder(t *testing.T) {
	t.Parallel()
	test.AssertEquals(t, findValue("Key (id)=(blah) already exists."), "blah")
	test.AssertEquals(t, findValue("Key (foo)=(foo blah) already exists."), "foo blah")
	test.AssertEquals(t, findValue("Unknown detail message"), "")
}
