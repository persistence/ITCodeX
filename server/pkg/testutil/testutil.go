package testutil

import (
	"testing"
)

// Require fails the test if err is not nil.
func Require(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
