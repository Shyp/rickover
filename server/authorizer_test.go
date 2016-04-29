package server

import (
	"testing"

	"github.com/Shyp/rickover/test"
)

func TestAddUserAuthsUser(t *testing.T) {
	AddUser("foo", "bar")
	err := DefaultAuthorizer.Authorize("foo", "bar")
	test.AssertNotError(t, err, "")

	err = DefaultAuthorizer.Authorize("foo", "wrongpassword")
	test.AssertError(t, err, "")
	test.AssertEquals(t, err.Error(), "Incorrect password for user foo")

	err = DefaultAuthorizer.Authorize("Unknownuser", "wrongpassword")
	test.AssertError(t, err, "")
	test.AssertEquals(t, err.Error(), "Username or password are invalid. Please double check your credentials")
}
