# Readable database errors

Do you like forcing constraint validation down into the database, but dislike
the native error messages returned by Postgres?

> invalid input syntax for uuid: "foo"

> invalid input value for enum account_status: "blah"

> new row for relation "accounts" violates check constraint "accounts_balance_check"

> null value in column \"id\" violates not-null constraint

This library attempts to parse those error messages and return error messages
that you can expose via your API.

> No id was provided. Please provide a id

> Cannot write a negative balance

> Invalid input syntax for type uuid: "foo"

> Invalid account_status: "blah"

> Can't save to payments because the account_id (91f47e99-d616-4d8c-9c02-cbd13bceac60) isn't present in the accounts table

> A email already exists with this value (test@example.com)

In addition, this library exports common Postgres error codes, so you can check
against them in your application.

## Basic Usage

```go
import dberror "github.com/Shyp/go-dberror"

func main() {
	_, err := db.Exec("INSERT INTO accounts (id) VALUES (null)")
	dberr := dberror.GetError(err)
	switch e := dberr.(type) {
	case *dberror.Error:
		fmt.Println(e.Error()) // "No id was provided. Please provide a id"
	default:
		// not a pq error
}
```

### Database Constraints

Failed check constraints are tricky - the native error messages just say
"failed", and don't reference a column.

So you can define your own constraint handlers, and then register them:

```go
import dberror "github.com/Shyp/go-dberror"
import "github.com/lib/pq"

func init()
	constraint := &dberror.Constraint{
		Name: "accounts_balance_check",
		GetError: func(e *pq.Error) *dberror.Error {
			return &dberror.Error{
				Message:  "Cannot write a negative balance",
				Severity: e.Severity,
				Table:    e.Table,
				Detail:   e.Detail,
				Code:     string(e.Code),
			}
		},
	}
	dberror.RegisterConstraint(constraint)

	// test.AssertEquals(t, e.Error(), "Cannot write a negative balance")
}
```

If we get a constraint failure, we'll call your `GetError` handler, to get
a well-formatted message.
