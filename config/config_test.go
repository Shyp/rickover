package config

import (
	"os"
	"reflect"
	"testing"

	"github.com/Shyp/rickover/test"
)

func TestVersionString(t *testing.T) {
	typ := reflect.TypeOf(Version)
	if typ.String() != "string" {
		t.Errorf("expected VERSION to be a string, got %#v (type %#v)", Version, typ.String())
	}
}

func TestGetInt(t *testing.T) {
	err := os.Setenv("CONFIG_TEST_INT_VAR", "5")
	test.AssertNotError(t, err, "setting env var")
	defer func() {
		os.Unsetenv("CONFIG_TEST_INT_VAR")
	}()
	i, err := GetInt("CONFIG_TEST_INT_VAR")
	test.AssertNotError(t, err, "getting env var")
	test.AssertEquals(t, i, 5)
}

func TestGetIntError(t *testing.T) {
	err := os.Setenv("CONFIG_TEST_INT_VAR", "bad")
	test.AssertNotError(t, err, "setting env var")
	defer func() {
		os.Unsetenv("CONFIG_TEST_INT_VAR")
	}()
	_, err = GetInt("CONFIG_TEST_INT_VAR")
	test.AssertError(t, err, "getting bad env var")
}
