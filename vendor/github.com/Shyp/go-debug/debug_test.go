package debug

import "testing"

import (
	"strings"
	"sync"
)
import "bytes"
import "time"

func assertContains(t *testing.T, str, substr string) {
	if !strings.Contains(str, substr) {
		t.Fatalf("expected %q to contain %q", str, substr)
	}
}

func assertNotContains(t *testing.T, str, substr string) {
	if strings.Contains(str, substr) {
		t.Fatalf("expected %q to not contain %q", str, substr)
	}
}

func TestDefault(t *testing.T) {
	var b []byte
	buf := bytes.NewBuffer(b)
	SetWriter(buf)

	debug := Debug("foo")
	debug("something")
	debug("here")
	debug("whoop")

	if buf.Len() != 0 {
		t.Fatalf("buffer should be empty")
	}
}

func TestEnable(t *testing.T) {
	var b []byte
	buf := bytes.NewBuffer(b)
	SetWriter(buf)

	Enable("foo")

	debug := Debug("foo")
	debug("something")
	debug("here")
	debug("whoop")

	if buf.Len() == 0 {
		t.Fatalf("buffer should have output")
	}

	str := string(buf.Bytes())
	assertContains(t, str, "something")
	assertContains(t, str, "here")
	assertContains(t, str, "whoop")
}

func TestMultipleOneEnabled(t *testing.T) {
	var b []byte
	buf := bytes.NewBuffer(b)
	SetWriter(buf)

	Enable("foo")

	foo := Debug("foo")
	foo("foo")

	bar := Debug("bar")
	bar("bar")

	if buf.Len() == 0 {
		t.Fatalf("buffer should have output")
	}

	str := string(buf.Bytes())
	assertContains(t, str, "foo")
	assertNotContains(t, str, "bar")
}

func TestMultipleEnabled(t *testing.T) {
	var b []byte
	buf := bytes.NewBuffer(b)
	SetWriter(buf)

	Enable("foo,bar")

	foo := Debug("foo")
	foo("foo")

	bar := Debug("bar")
	bar("bar")

	if buf.Len() == 0 {
		t.Fatalf("buffer should have output")
	}

	str := string(buf.Bytes())
	assertContains(t, str, "foo")
	assertContains(t, str, "bar")
}

func TestEnableDisable(t *testing.T) {
	var b []byte
	buf := bytes.NewBuffer(b)
	SetWriter(buf)

	Enable("foo,bar")
	Disable()

	foo := Debug("foo")
	foo("foo")

	bar := Debug("bar")
	bar("bar")

	if buf.Len() != 0 {
		t.Fatalf("buffer should not have output")
	}
}

// TestRace helps reveal race conditions. Use it like this:
// go test -v -run TestRace -race
func TestRace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping")
	}

	var b []byte
	buf := bytes.NewBuffer(b)
	SetWriter(buf)

	Enable("foo")
	foo := Debug("foo")

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for d := 0; d < 100; d++ {
				foo("f")
			}
		}()
	}

	wg.Wait()
}

func ExampleEnable() {
	Enable("mongo:connection")
	Enable("mongo:*")
	Enable("foo,bar,baz")
	Enable("*")
}

func ExampleDebug() {
	var debug = Debug("single")

	for {
		debug("sending mail")
		debug("send email to %s", "tobi@segment.io")
		debug("send email to %s", "loki@segment.io")
		debug("send email to %s", "jane@segment.io")
		time.Sleep(500 * time.Millisecond)
	}
}

func BenchmarkEnabled(b *testing.B) {
	var out []byte
	buf := bytes.NewBuffer(out)
	SetWriter(buf)

	Enable("foo")
	foo := Debug("foo")
	for i := 0; i < b.N; i++ {
		foo("stuff")
	}
}

func BenchmarkDisabled(b *testing.B) {
	debug := Debug("something")
	for i := 0; i < b.N; i++ {
		debug("stuff")
	}
}

func BenchmarkNonMatch(b *testing.B) {
	debug := Debug("something")
	Enable("nonmatch")
	for i := 0; i < b.N; i++ {
		debug("stuff")
	}
}
