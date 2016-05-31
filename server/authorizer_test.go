package server

import (
	"testing"

	"github.com/Shyp/rickover/test"
)

func TestAddUserAuthsUser(t *testing.T) {
	t.Parallel()
	AddUser("test_adduser_foo", "bar")
	err := DefaultAuthorizer.Authorize("test_adduser_foo", "bar")
	test.Assert(t, err == nil, "")

	err = DefaultAuthorizer.Authorize("test_adduser_foo", "wrongpassword")
	test.AssertError(t, err, "")
	test.AssertEquals(t, err.Error(), "Incorrect password for user test_adduser_foo")

	err = DefaultAuthorizer.Authorize("Unknownuser", "wrongpassword")
	test.AssertError(t, err, "")
	test.AssertEquals(t, err.Error(), "Username or password are invalid. Please double check your credentials")
}
