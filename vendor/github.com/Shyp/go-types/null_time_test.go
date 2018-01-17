package types

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func ExampleNullTime() {
	t, _ := time.Parse(time.RFC3339, "2016-05-02T09:03:04-07:00")
	nt := NullTime{Valid: true, Time: t}
	json.NewEncoder(os.Stdout).Encode(nt)
	// Output: "2016-05-02T09:03:04-07:00"
}

func TestTime(t *testing.T) {
	var nt NullTime
	str := []byte("\"2015-08-03T22:43:19.000Z\"")
	err := json.Unmarshal(str, &nt)
	assertNotError(t, err, "")
	assertEquals(t, nt.Valid, true)
	assertEquals(t, nt.Time.Year(), 2015)
	assertEquals(t, nt.Time.Second(), 19)
}

func TestNullTime(t *testing.T) {
	var nt NullTime
	str := []byte("null")
	err := json.Unmarshal(str, &nt)
	assertNotError(t, err, "")
	assertEquals(t, nt.Valid, false)
}

func TestNullTimeMarshal(t *testing.T) {
	tim, _ := time.Parse("2006-01-02", "2016-01-01")
	nt := NullTime{
		Valid: true,
		Time:  tim,
	}
	bits, err := json.Marshal(nt)
	assertNotError(t, err, "")
	assertEquals(t, string(bits), "\"2016-01-01T00:00:00Z\"")
}

func TestNullTimeNullMarshal(t *testing.T) {
	nt := NullTime{
		Valid: false,
		Time:  time.Time{},
	}
	bits, err := json.Marshal(nt)
	assertNotError(t, err, "")
	assertEquals(t, string(bits), "null")
}
