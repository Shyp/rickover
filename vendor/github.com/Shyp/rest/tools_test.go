// Copyright 2014 ISRG.  All rights reserved
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Copied from https://github.com/letsencrypt/boulder/blob/master/test/test-tools.go
//
// See Q5 and Q11 here: https://www.mozilla.org/en-US/MPL/2.0/FAQ/ I think if
// we want to modify this file we have to release it publicly, otherwise we're
// fine.

package rest

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

// Return short format caller info for printing errors, so errors don't all
// appear to come from test-tools.go.
func caller() string {
	_, file, line, _ := runtime.Caller(2)
	splits := strings.Split(file, "/")
	filename := splits[len(splits)-1]
	return fmt.Sprintf("%s:%d:", filename, line)
}

// Assert a boolean
func assert(t *testing.T, result bool, message string) {
	if !result {
		t.Fatalf("%s %s", caller(), message)
	}
}

// AssertNotError checks that err is nil
func assertNotError(t *testing.T, err error, message string) {
	if err != nil {
		t.Fatalf("%s %s: %s", caller(), message, err)
	}
}

// AssertError checks that err is non-nil
func assertError(t *testing.T, err error, message string) {
	if err == nil {
		t.Fatalf("%s %s: expected error but received none", caller(), message)
	}
}

// AssertEquals uses the equality operator (==) to measure one and two
func assertEquals(t *testing.T, one interface{}, two interface{}) {
	if one != two {
		t.Fatalf("%s [%v] != [%v]", caller(), one, two)
	}
}

// assertContains determines whether needle can be found in haystack
func assertContains(t *testing.T, haystack string, needle string) {
	if !strings.Contains(haystack, needle) {
		t.Fatalf("%s String [%s] does not contain [%s]", caller(), haystack, needle)
	}
}
