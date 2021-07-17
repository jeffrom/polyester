package state

import (
	"testing"
)

type KVI = map[string]interface{}

func TestStateChange(t *testing.T) {
	tcs := []struct {
		name         string
		a            State
		b            State
		expectChange bool
	}{
		{
			name: "empty",
		},
		{
			name:         "kv-basic",
			a:            fromEntries(kviEntry("a", KVI{"attr": true})),
			b:            fromEntries(kviEntry("b", KVI{"attr": true})),
			expectChange: true,
		},
		{
			name: "kv-basic-same",
			a:    fromEntries(kviEntry("a", KVI{"attr": true})),
			b:    fromEntries(kviEntry("a", KVI{"attr": true})),
		},
		{
			name:         "kv-empty",
			a:            fromEntries(kviEntry("a", KVI{"attr": true})),
			b:            fromEntries(kviEntry("a", nil)),
			expectChange: true,
		},
		{
			name:         "kv-type",
			a:            fromEntries(kviEntry("a", KVI{"attr": true})),
			b:            fromEntries(kviEntry("a", KVI{"attr": "true"})),
			expectChange: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			changed := tc.a.Changed(tc.b)
			if tc.expectChange && !changed {
				t.Error("expected change but got none")
			} else if !tc.expectChange && changed {
				t.Error("expected no change but got one")
			}
		})
	}
}

func kviEntry(name string, kvi KVI) Entry { return Entry{Name: name, KV: kvi} }

func fromEntries(entries ...Entry) State { return State{Entries: entries} }

// func fromString(t testing.TB, s string) State {
// 	t.Helper()

// 	b := []byte(fmt.Sprintf(`{"entries": %s}`, s))
// 	st, err := FromBytes(b)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	return st
// }
