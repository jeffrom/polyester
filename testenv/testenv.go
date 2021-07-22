// Package testenv contains testing helpers for creating test harnesses and
// other common tasks.
package testenv

import "testing"

func die(err error) {
	if err != nil {
		panic(err)
	}
}

func logError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Logf("testenv: unhandled error: %+v", err)
	}
}
