package types

import (
	"encoding/json"
	"os"
	"testing"
)

func ExampleNullString() {
	n := NullString{Valid: true, String: "foo"}
	json.NewEncoder(os.Stdout).Encode(n)
	// Output: "foo"
}

func TestString(t *testing.T) {
	var ns NullString
	str := []byte("\"foo\"")
	err := json.Unmarshal(str, &ns)
	assertNotError(t, err, "")
	assertEquals(t, ns.Valid, true)
	assertEquals(t, ns.String, "foo")
}

func TestNullString(t *testing.T) {
	var ns NullString
	str := []byte("null")
	err := json.Unmarshal(str, &ns)
	assertNotError(t, err, "")
	assertEquals(t, ns.Valid, false)
}

func TestStringMarshal(t *testing.T) {
	ns := NullString{
		Valid:  true,
		String: "foo bar",
	}
	b, err := json.Marshal(ns)
	assertNotError(t, err, "")
	assertEquals(t, string(b), "\"foo bar\"")
}

func TestStringMarshalNull(t *testing.T) {
	ns := NullString{
		Valid:  false,
		String: "",
	}
	b, err := json.Marshal(ns)
	assertNotError(t, err, "")
	assertEquals(t, string(b), "null")
}
