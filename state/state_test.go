package state

import (
	"fmt"
	"testing"
)

func TestState(t *testing.T) {
	a := fromString(t, `[{"name": "a", "kv": {"a": "a"}}]`)
	if a.Changed(a) {
		t.Error("expected no change (a -> a)")
	}
	b := fromString(t, `[{"name": "b", "kv": {"b": "b"}}]`)
	if !a.Changed(b) {
		t.Error("expected change (a -> b)")
	}
}

func fromString(t testing.TB, s string) State {
	t.Helper()

	b := []byte(fmt.Sprintf(`{"entries": %s}`, s))
	st, err := FromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	return st
}
