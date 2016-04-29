package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/Shyp/rickover/test"
)

func TestRunAfter(t *testing.T) {
	ra := getRunAfter(12, 12)
	diff := time.Second - ra.Sub(time.Now())
	test.Assert(t, diff < 2*time.Millisecond, fmt.Sprint(diff))

	ra = getRunAfter(12, 11)
	diff = 2*time.Second - ra.Sub(time.Now())
	test.Assert(t, diff < 2*time.Millisecond, fmt.Sprint(diff))

	ra = getRunAfter(12, 10)
	diff = 4*time.Second - ra.Sub(time.Now())
	test.Assert(t, diff < 2*time.Millisecond, fmt.Sprint(diff))

	ra = getRunAfter(12, 1)
	diff = 2048*time.Second - ra.Sub(time.Now())
	test.Assert(t, diff < 2*time.Millisecond, fmt.Sprint(diff))
}
